package function

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-crm/internal/features/email_template"
	"go-crm/internal/features/record"
)

const (
	MaxArrayItems = 10
	MaxStringLen  = 500
)

type FunctionService interface {
	CreateFunction(ctx context.Context, function *Function) error
	GetFunction(ctx context.Context, id string) (*Function, error)
	ListFunctions(ctx context.Context, moduleName string, activeOnly bool) ([]Function, error)
	UpdateFunction(ctx context.Context, function *Function) error
	DeleteFunction(ctx context.Context, id string) error
	ExecuteFunction(ctx context.Context, functionID string, moduleName string, record map[string]any) error
	TestFunction(ctx context.Context, functionID string, testData map[string]any, codeOverride string) (any, error)
	ValidateCode(language FunctionLanguage, code string) error
}

type FunctionServiceImpl struct {
	repo                 FunctionRepository
	recordRepo           record.RecordRepository
	emailTemplateService email_template.EmailTemplateService
}

func NewFunctionService(repo FunctionRepository, recordRepo record.RecordRepository, emailTemplateService email_template.EmailTemplateService) FunctionService {
	return &FunctionServiceImpl{
		repo:                 repo,
		recordRepo:           recordRepo,
		emailTemplateService: emailTemplateService,
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

func (s *FunctionServiceImpl) ExecuteFunction(ctx context.Context, functionID string, moduleName string, record map[string]any) error {
	function, err := s.repo.GetByID(ctx, functionID)
	if err != nil {
		return fmt.Errorf("failed to get function: %w", err)
	}

	if !function.IsActive {
		return fmt.Errorf("function is not active")
	}

	if function.Language != LanguageJavaScript {
		return fmt.Errorf("unsupported language: %s", function.Language)
	}

	return s.executeJavaScript(ctx, function.Code, moduleName, record, function.Parameters)
}

func (s *FunctionServiceImpl) TestFunction(ctx context.Context, functionID string, testData map[string]any, codeOverride string) (any, error) {
	function, err := s.repo.GetByID(ctx, functionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	codeToRun := function.Code
	if codeOverride != "" {
		codeToRun = codeOverride
	}

	if function.Language != LanguageJavaScript {
		return nil, fmt.Errorf("unsupported language: %s", function.Language)
	}

	return s.testJavaScript(ctx, codeToRun, testData)
}

func (s *FunctionServiceImpl) ValidateCode(language FunctionLanguage, code string) error {
	if language != LanguageJavaScript {
		return fmt.Errorf("unsupported language: %s", language)
	}

	vm := goja.New()
	// Add dummy context variables for validation
	vm.Set("module", "")
	vm.Set("record", map[string]any{})
	vm.Set("args", map[string]any{})
	// Add console for validation
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	vm.Set("console", console)

	// Add crm for validation
	crm := vm.NewObject()
	crm.Set("find", func(call goja.FunctionCall) goja.Value { return vm.ToValue([]map[string]any{}) })
	crm.Set("first", func(call goja.FunctionCall) goja.Value { return vm.ToValue(map[string]any{"id": "mock_id"}) })
	crm.Set("create", func(call goja.FunctionCall) goja.Value { return vm.ToValue("mock_id") })
	crm.Set("update", func(call goja.FunctionCall) goja.Value { return vm.ToValue(true) })
	crm.Set("delete", func(call goja.FunctionCall) goja.Value { return vm.ToValue(true) })
	vm.Set("crm", crm)

	// Add http/email for validation
	httpObj := vm.NewObject()

	// Create a hybrid mock data object that has both array methods and object properties
	// This satisfies validation for code expecting either an Array or an Object
	// We use an Array as the base so it is iterable and has array methods by default
	mockData := vm.NewArray()
	// Add a dummy item so iteration works
	mockData.Set("0", map[string]any{"id": 1, "name": "Mock Item"})
	mockData.Set("length", 1)

	// Add common object properties to the array instance
	mockData.Set("id", "mock_id")
	mockData.Set("name", "mock_name")
	mockData.Set("status", "active")

	// We don't need to manually set map/filter etc because NewArray() provides them
	// but we can add 'find' if it's missing or specific object methods if needed.
	// Arrays in JS are just objects, so this approach is valid.

	mockResp := map[string]any{
		"status":  200,
		"headers": map[string]any{},
		"data":    mockData,
	}
	httpObj.Set("get", func(call goja.FunctionCall) goja.Value { return vm.ToValue(mockResp) })
	httpObj.Set("post", func(call goja.FunctionCall) goja.Value { return vm.ToValue(mockResp) })
	httpObj.Set("put", func(call goja.FunctionCall) goja.Value { return vm.ToValue(mockResp) })
	httpObj.Set("delete", func(call goja.FunctionCall) goja.Value { return vm.ToValue(mockResp) })
	vm.Set("http", httpObj)

	emailObj := vm.NewObject()
	emailObj.Set("send", func(call goja.FunctionCall) goja.Value { return vm.ToValue(true) })
	vm.Set("email", emailObj)

	_, err := vm.RunString(code)
	return err
}

func (s *FunctionServiceImpl) executeJavaScript(ctx context.Context, code string, moduleName string, record map[string]any, parameters []FunctionParameter) error {
	vm := goja.New()

	// Map arguments from record based on parameter names
	args := make(map[string]any)
	for _, param := range parameters {
		// Special handling for record_id
		if param.Name == "record_id" {
			if id, ok := record["_id"]; ok {
				if idObj, ok := id.(primitive.ObjectID); ok {
					args[param.Name] = idObj.Hex()
				} else {
					args[param.Name] = fmt.Sprintf("%v", id)
				}
			}
			continue
		}

		// Direct mapping by name
		if val, ok := record[param.Name]; ok {
			args[param.Name] = val
		}
	}

	// Set context variables
	vm.Set("module", moduleName)
	vm.Set("record", record)
	vm.Set("args", args)

	// Add console.log support
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		var args []any
		for _, arg := range call.Arguments {
			args = append(args, arg.Export())
		}
		log.Println(args...)
		return goja.Undefined()
	})
	vm.Set("console", console)

	// Add helpers
	s.setupJavaScriptDB(vm, ctx)
	s.setupJavaScriptHTTP(vm)
	s.setupJavaScriptEmail(vm, ctx)

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
		filter := make(map[string]any)
		if len(call.Arguments) > 1 {
			if f, ok := call.Argument(1).Export().(map[string]any); ok {
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
		data, ok := call.Argument(1).Export().(map[string]any)
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
		data, ok := call.Argument(2).Export().(map[string]any)
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

func (s *FunctionServiceImpl) testJavaScript(ctx context.Context, code string, testData map[string]any) (any, error) {
	vm := goja.New()

	// Set test data
	for key, value := range testData {
		vm.Set(key, value)
	}

	// Add console.log support with line numbers
	var logs []string
	console := vm.NewObject()

	console.Set("log", func(call goja.FunctionCall) goja.Value {
		var out []string

		for _, arg := range call.Arguments {
			exported := truncateValue(arg.Export())

			// Try JSON first (covers map, slice, struct, nested)
			if b, err := json.MarshalIndent(exported, "", "  "); err == nil {
				out = append(out, string(b))
			} else {
				out = append(out, fmt.Sprintf("%v", exported))
			}
		}

		msg := strings.Join(out, " ")

		// Line number
		if stack := vm.CaptureCallStack(10, nil); len(stack) > 1 {
			msg += fmt.Sprintf(" (line %d)", stack[1].Position().Line)
		}

		logs = append(logs, msg)
		return goja.Undefined()
	})

	vm.Set("console", console)

	// Add helpers
	s.setupJavaScriptDB(vm, ctx)
	s.setupJavaScriptHTTP(vm)
	s.setupJavaScriptEmail(vm, ctx)

	var runErr error
	result, err := vm.RunString(code)
	if err != nil {
		if jsErr, ok := err.(*goja.Exception); ok {
			// Capture the error but don't fail the request (so we get logs)
			runErr = fmt.Errorf("%s", jsErr.String())
			logs = append(logs, fmt.Sprintf("Error: %v", runErr))
		} else {
			return nil, fmt.Errorf("failed to run JavaScript: %w", err)
		}
	}

	response := map[string]any{
		"result":  nil,
		"logs":    logs,
		"success": runErr == nil,
	}

	if result != nil {
		response["result"] = result.Export()
	}

	if runErr != nil {
		response["error"] = runErr.Error()
	}

	return response, nil
}

// Helper methods for HTTP and Email support

func (s *FunctionServiceImpl) setupJavaScriptHTTP(vm *goja.Runtime) {
	httpObj := vm.NewObject()
	client := &http.Client{Timeout: 30 * time.Second}

	doRequest := func(method string, call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return goja.Undefined()
		}
		urlStr := call.Argument(0).String()

		var bodyReader io.Reader
		var headers map[string]string

		// Args for post/put: url, body, headers
		// Args for get/delete: url, headers

		if method == "POST" || method == "PUT" {
			if len(call.Arguments) > 1 {
				bodyVal := call.Argument(1).Export()
				if bodyVal != nil {
					jsonBody, _ := json.Marshal(bodyVal)
					bodyReader = bytes.NewBuffer(jsonBody)
				}
			}
			if len(call.Arguments) > 2 {
				if h, ok := call.Argument(2).Export().(map[string]any); ok {
					headers = make(map[string]string)
					for k, v := range h {
						headers[k] = fmt.Sprint(v)
					}
				}
			}
		} else {
			if len(call.Arguments) > 1 {
				if h, ok := call.Argument(1).Export().(map[string]any); ok {
					headers = make(map[string]string)
					for k, v := range h {
						headers[k] = fmt.Sprint(v)
					}
				}
			}
		}

		req, err := http.NewRequest(method, urlStr, bodyReader)
		if err != nil {
			panic(vm.ToValue(fmt.Sprintf("failed to create request: %v", err)))
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if bodyReader != nil && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err != nil {
			panic(vm.ToValue(fmt.Sprintf("request failed: %v", err)))
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)

		var jsonResp any
		// Try to parse JSON response
		if err := json.Unmarshal(respBody, &jsonResp); err != nil {
			// If not JSON, return string
			jsonResp = string(respBody)
		}

		result := map[string]any{
			"status":  resp.StatusCode,
			"headers": resp.Header,
			"data":    jsonResp,
		}

		return vm.ToValue(result)
	}

	httpObj.Set("get", func(call goja.FunctionCall) goja.Value {
		return doRequest("GET", call)
	})
	httpObj.Set("post", func(call goja.FunctionCall) goja.Value {
		return doRequest("POST", call)
	})
	httpObj.Set("put", func(call goja.FunctionCall) goja.Value {
		return doRequest("PUT", call)
	})
	httpObj.Set("delete", func(call goja.FunctionCall) goja.Value {
		return doRequest("DELETE", call)
	})

	vm.Set("http", httpObj)
}

func (s *FunctionServiceImpl) setupJavaScriptEmail(vm *goja.Runtime, ctx context.Context) {
	emailObj := vm.NewObject()

	emailObj.Set("send", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.ToValue("email.send requires 3 arguments: templateId, to, data"))
		}

		templateID := call.Argument(0).String()
		to := call.Argument(1).String()
		data, ok := call.Argument(2).Export().(map[string]any)
		if !ok {
			panic(vm.ToValue("data must be an object"))
		}

		err := s.emailTemplateService.SendTestEmail(ctx, templateID, to, data)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}

		return vm.ToValue(true)
	})

	vm.Set("email", emailObj)
}

func truncateValue(v any) any {
	switch val := v.(type) {

	case []any:
		if len(val) > MaxArrayItems {
			return append(
				val[:MaxArrayItems],
				fmt.Sprintf("... (%d more items)", len(val)-MaxArrayItems),
			)
		}
		return val

	case map[string]any:
		out := make(map[string]any)
		for k, v2 := range val {
			out[k] = truncateValue(v2)
		}
		return out

	case string:
		if len(val) > MaxStringLen {
			return val[:MaxStringLen] + "... (truncated)"
		}
		return val

	default:
		return val
	}
}
