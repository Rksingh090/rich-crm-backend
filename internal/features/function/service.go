package function

import (
	"context"
	"fmt"
	"log"

	"go-crm/internal/features/record"

	"github.com/d5/tengo/v2"
	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FunctionService interface {
	CreateFunction(ctx context.Context, function *Function) error
	GetFunction(ctx context.Context, id string) (*Function, error)
	ListFunctions(ctx context.Context, moduleName string, activeOnly bool) ([]Function, error)
	UpdateFunction(ctx context.Context, function *Function) error
	DeleteFunction(ctx context.Context, id string) error
	ExecuteFunction(ctx context.Context, functionID string, moduleName string, record map[string]interface{}) error
	TestFunction(ctx context.Context, functionID string, testData map[string]interface{}) (interface{}, error)
	ValidateCode(language FunctionLanguage, code string) error
}

type FunctionServiceImpl struct {
	repo       FunctionRepository
	recordRepo record.RecordRepository
}

func NewFunctionService(repo FunctionRepository, recordRepo record.RecordRepository) FunctionService {
	return &FunctionServiceImpl{
		repo:       repo,
		recordRepo: recordRepo,
	}
}

func (s *FunctionServiceImpl) CreateFunction(ctx context.Context, function *Function) error {
	// Validate code syntax
	if err := s.ValidateCode(function.Language, function.Code); err != nil {
		return fmt.Errorf("code validation failed: %w", err)
	}

	return s.repo.Create(ctx, function)
}

func (s *FunctionServiceImpl) GetFunction(ctx context.Context, id string) (*Function, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *FunctionServiceImpl) ListFunctions(ctx context.Context, moduleName string, activeOnly bool) ([]Function, error) {
	functions, err := s.repo.List(ctx, moduleName)
	if err != nil {
		return nil, err
	}

	if activeOnly {
		var activeFunctions []Function
		for _, fn := range functions {
			if fn.IsActive {
				activeFunctions = append(activeFunctions, fn)
			}
		}
		return activeFunctions, nil
	}

	return functions, nil
}

func (s *FunctionServiceImpl) UpdateFunction(ctx context.Context, function *Function) error {
	// Validate code syntax
	if err := s.ValidateCode(function.Language, function.Code); err != nil {
		return fmt.Errorf("code validation failed: %w", err)
	}

	return s.repo.Update(ctx, function)
}

func (s *FunctionServiceImpl) DeleteFunction(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *FunctionServiceImpl) ExecuteFunction(ctx context.Context, functionID string, moduleName string, record map[string]interface{}) error {
	function, err := s.repo.GetByID(ctx, functionID)
	if err != nil {
		return fmt.Errorf("failed to get function: %w", err)
	}

	if !function.IsActive {
		return fmt.Errorf("function is not active")
	}

	switch function.Language {
	case LanguageTengo:
		return s.executeTengo(ctx, function.Code, moduleName, record)
	case LanguageJavaScript:
		return s.executeJavaScript(ctx, function.Code, moduleName, record)
	default:
		return fmt.Errorf("unsupported language: %s", function.Language)
	}
}

func (s *FunctionServiceImpl) TestFunction(ctx context.Context, functionID string, testData map[string]interface{}) (interface{}, error) {
	function, err := s.repo.GetByID(ctx, functionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	switch function.Language {
	case LanguageTengo:
		return s.testTengo(ctx, function.Code, testData)
	case LanguageJavaScript:
		return s.testJavaScript(ctx, function.Code, testData)
	default:
		return nil, fmt.Errorf("unsupported language: %s", function.Language)
	}
}

func (s *FunctionServiceImpl) ValidateCode(language FunctionLanguage, code string) error {
	switch language {
	case LanguageTengo:
		script := tengo.NewScript([]byte(code))
		// Add dummy context variables for validation
		script.Add("module", "")
		script.Add("record", map[string]interface{}{})
		script.Add("args", map[string]interface{}{})

		// Add CRM functions for validation
		script.Add("crm_find", func(args ...tengo.Object) (tengo.Object, error) { return tengo.UndefinedValue, nil })
		script.Add("crm_first", func(args ...tengo.Object) (tengo.Object, error) { return tengo.UndefinedValue, nil })
		script.Add("crm_create", func(args ...tengo.Object) (tengo.Object, error) { return tengo.UndefinedValue, nil })
		script.Add("crm_update", func(args ...tengo.Object) (tengo.Object, error) { return tengo.UndefinedValue, nil })
		script.Add("crm_delete", func(args ...tengo.Object) (tengo.Object, error) { return tengo.UndefinedValue, nil })

		_, err := script.Compile()
		return err
	case LanguageJavaScript:
		vm := goja.New()
		// Add dummy context variables for validation
		vm.Set("module", "")
		vm.Set("record", map[string]interface{}{})
		vm.Set("args", map[string]interface{}{})
		// Add console for validation
		console := vm.NewObject()
		console.Set("log", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		vm.Set("console", console)

		// Add crm for validation
		crm := vm.NewObject()
		crm.Set("find", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		crm.Set("first", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		crm.Set("create", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		crm.Set("update", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		crm.Set("delete", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
		vm.Set("crm", crm)

		_, err := vm.RunString(code)
		return err
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

func (s *FunctionServiceImpl) executeTengo(ctx context.Context, code string, moduleName string, record map[string]interface{}) error {
	script := tengo.NewScript([]byte(code))

	// Add context variables
	script.Add("module", moduleName)
	script.Add("record", record)
	script.Add("args", map[string]interface{}{}) // Empty args for now

	// Add CRM helpers
	s.setupTengoCRM(script, ctx)

	compiled, err := script.Compile()
	if err != nil {
		return fmt.Errorf("failed to compile Tengo script: %w", err)
	}

	if err := compiled.Run(); err != nil {
		return fmt.Errorf("failed to run Tengo script: %w", err)
	}

	log.Printf("Executed Tengo function for module %s", moduleName)
	return nil
}

func (s *FunctionServiceImpl) executeJavaScript(ctx context.Context, code string, moduleName string, record map[string]interface{}) error {
	vm := goja.New()

	// Set context variables
	vm.Set("module", moduleName)
	vm.Set("record", record)
	vm.Set("args", map[string]interface{}{}) // Empty args for now

	// Add console.log support
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		var args []interface{}
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		log.Println(args...)
		return goja.Undefined()
	})
	vm.Set("console", console)

	// Add DB helper
	s.setupJavaScriptDB(vm, ctx)

	_, err := vm.RunString(code)
	if err != nil {
		return fmt.Errorf("failed to run JavaScript: %w", err)
	}

	log.Printf("Executed JavaScript function for module %s", moduleName)
	return nil
}

func (s *FunctionServiceImpl) setupJavaScriptDB(vm *goja.Runtime, ctx context.Context) {
	db := vm.NewObject()

	db.Set("find", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		module := call.Argument(0).String()
		filter := make(map[string]interface{})
		if len(call.Arguments) > 1 {
			if f, ok := call.Argument(1).Export().(map[string]interface{}); ok {
				filter = f
			}
		}

		records, err := s.recordRepo.List(ctx, module, filter, nil, 100, 0, "", -1)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		return vm.ToValue(records)
	})

	db.Set("first", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		module := call.Argument(0).String()
		id := call.Argument(1).String()

		rec, err := s.recordRepo.Get(ctx, module, id)
		if err != nil {
			return goja.Null()
		}
		return vm.ToValue(rec)
	})

	db.Set("create", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		module := call.Argument(0).String()
		data, ok := call.Argument(1).Export().(map[string]interface{})
		if !ok {
			panic(vm.ToValue("data must be an object"))
		}

		id, err := s.recordRepo.Create(ctx, module, "", data)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		return vm.ToValue(id)
	})

	db.Set("update", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			return goja.Undefined()
		}
		module := call.Argument(0).String()
		id := call.Argument(1).String()
		data, ok := call.Argument(2).Export().(map[string]interface{})
		if !ok {
			panic(vm.ToValue("data must be an object"))
		}

		err := s.recordRepo.Update(ctx, module, id, data)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		return vm.ToValue(true)
	})

	db.Set("delete", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		module := call.Argument(0).String()
		id := call.Argument(1).String()

		err := s.recordRepo.Delete(ctx, module, id, primitive.NilObjectID)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		return vm.ToValue(true)
	})

	vm.Set("crm", db)
}

func (s *FunctionServiceImpl) setupTengoCRM(script *tengo.Script, ctx context.Context) {
	script.Add("crm_find", func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return tengo.UndefinedValue, nil
		}
		module, ok := tengo.ToString(args[0])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "module", Expected: "string", Found: args[0].TypeName()}
		}

		filter := make(map[string]interface{})
		if len(args) > 1 {
			if f, ok := tengo.ToInterface(args[1]).(map[string]interface{}); ok {
				filter = f
			}
		}

		records, err := s.recordRepo.List(ctx, module, filter, nil, 100, 0, "", -1)
		if err != nil {
			return nil, err
		}

		obj, err := tengo.FromInterface(records)
		return obj, err
	})

	script.Add("crm_first", func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 2 {
			return tengo.UndefinedValue, nil
		}
		module, ok := tengo.ToString(args[0])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "module", Expected: "string", Found: args[0].TypeName()}
		}
		id, ok := tengo.ToString(args[1])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "id", Expected: "string", Found: args[1].TypeName()}
		}

		rec, err := s.recordRepo.Get(ctx, module, id)
		if err != nil {
			return tengo.UndefinedValue, nil
		}

		obj, err := tengo.FromInterface(rec)
		return obj, err
	})

	script.Add("crm_create", func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 2 {
			return tengo.UndefinedValue, nil
		}
		module, ok := tengo.ToString(args[0])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "module", Expected: "string", Found: args[0].TypeName()}
		}
		data, ok := tengo.ToInterface(args[1]).(map[string]interface{})
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "data", Expected: "map", Found: args[1].TypeName()}
		}

		id, err := s.recordRepo.Create(ctx, module, "", data)
		if err != nil {
			return nil, err
		}

		obj, err := tengo.FromInterface(id)
		return obj, err
	})

	script.Add("crm_update", func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 3 {
			return tengo.UndefinedValue, nil
		}
		module, ok := tengo.ToString(args[0])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "module", Expected: "string", Found: args[0].TypeName()}
		}
		id, ok := tengo.ToString(args[1])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "id", Expected: "string", Found: args[1].TypeName()}
		}
		data, ok := tengo.ToInterface(args[2]).(map[string]interface{})
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "data", Expected: "map", Found: args[2].TypeName()}
		}

		err := s.recordRepo.Update(ctx, module, id, data)
		if err != nil {
			return nil, err
		}

		return tengo.TrueValue, nil
	})

	script.Add("crm_delete", func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 2 {
			return tengo.UndefinedValue, nil
		}
		module, ok := tengo.ToString(args[0])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "module", Expected: "string", Found: args[0].TypeName()}
		}
		id, ok := tengo.ToString(args[1])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "id", Expected: "string", Found: args[1].TypeName()}
		}

		err := s.recordRepo.Delete(ctx, module, id, primitive.NilObjectID)
		if err != nil {
			return nil, err
		}

		return tengo.TrueValue, nil
	})
}

func (s *FunctionServiceImpl) testTengo(ctx context.Context, code string, testData map[string]interface{}) (interface{}, error) {
	script := tengo.NewScript([]byte(code))

	// Add test data
	for key, value := range testData {
		script.Add(key, value)
	}

	// Add CRM helpers
	s.setupTengoCRM(script, ctx)

	compiled, err := script.Compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile Tengo script: %w", err)
	}

	if err := compiled.Run(); err != nil {
		return nil, fmt.Errorf("failed to run Tengo script: %w", err)
	}

	// Get result variable if it exists
	if result := compiled.Get("result"); result != nil {
		return result.Value(), nil
	}

	return "Script executed successfully", nil
}

func (s *FunctionServiceImpl) testJavaScript(ctx context.Context, code string, testData map[string]interface{}) (interface{}, error) {
	vm := goja.New()

	// Set test data
	for key, value := range testData {
		vm.Set(key, value)
	}

	// Add console.log support with line numbers
	var logs []string
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		var args []interface{}
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		logMsg := fmt.Sprint(args...)

		// Try to get stack trace to find line number
		lineInfo := ""
		if stack := vm.CaptureCallStack(10, nil); len(stack) > 0 {
			// The first frame is usually the console.log call itself
			// We want the caller of console.log
			if len(stack) > 1 {
				frame := stack[1]
				lineInfo = fmt.Sprintf(":%d", frame.Position().Line)
			}
		}

		logs = append(logs, logMsg+lineInfo)
		return goja.Undefined()
	})
	vm.Set("console", console)

	// Add DB helper
	s.setupJavaScriptDB(vm, ctx)

	result, err := vm.RunString(code)
	if err != nil {
		return nil, fmt.Errorf("failed to run JavaScript: %w", err)
	}

	response := map[string]interface{}{
		"result": result.Export(),
		"logs":   logs,
	}

	return response, nil
}
