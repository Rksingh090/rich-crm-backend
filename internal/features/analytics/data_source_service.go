package analytics

import (
	"context"
	"fmt"
	common_models "go-crm/internal/common/models"
	"go-crm/internal/connectors"
	"go-crm/internal/features/audit"
	"go-crm/internal/features/module"
	"go-crm/internal/features/record"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DataSourceService interface {
	// Data Source Management
	CreateDataSource(ctx context.Context, ds *DataSource) error
	GetDataSource(ctx context.Context, id string) (*DataSource, error)
	ListDataSources(ctx context.Context) ([]DataSource, error)
	UpdateDataSource(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteDataSource(ctx context.Context, id string) error
	TestDataSource(ctx context.Context, id string) error

	// Data Querying
	QueryDataSource(ctx context.Context, req connectors.QueryRequest) (*connectors.QueryResponse, error)
	GetDataSourceSchema(ctx context.Context, sourceID string, module string) (*connectors.SchemaInfo, error)

	// Cross-source queries
	QueryMultipleSources(ctx context.Context, queries []connectors.QueryRequest) (map[string]*connectors.QueryResponse, error)
}

type DataSourceServiceImpl struct {
	dataSourceRepo DataSourceRepository
	connectors     map[string]connectors.Connector
	crmConnector   connectors.Connector
	recordService  record.RecordService
	moduleService  module.ModuleService
	auditService   audit.AuditService
}

func NewDataSourceService(
	dataSourceRepo DataSourceRepository,
	recordService record.RecordService,
	moduleService module.ModuleService,
	auditService audit.AuditService,
) DataSourceService {
	// Create adapters for the connector interfaces
	recordProvider := &recordServiceAdapter{recordService}
	moduleProvider := &moduleServiceAdapter{moduleService}

	return &DataSourceServiceImpl{
		dataSourceRepo: dataSourceRepo,
		connectors:     make(map[string]connectors.Connector),
		crmConnector:   connectors.NewCRMConnector(recordProvider, moduleProvider),
		recordService:  recordService,
		moduleService:  moduleService,
		auditService:   auditService,
	}
}

// recordServiceAdapter adapts record.RecordService to connectors.RecordProvider
type recordServiceAdapter struct {
	service record.RecordService
}

func (a *recordServiceAdapter) ListRecords(ctx context.Context, moduleName string, filters map[string]any, limit int64, offset int64, sortField string, sortOrder string, userID primitive.ObjectID) ([]map[string]any, int64, error) {
	page := (offset / limit) + 1
	return a.service.ListRecords(ctx, moduleName, filters, page, limit, sortField, sortOrder, userID)
}

// moduleServiceAdapter adapts module.ModuleService to connectors.ModuleProvider
type moduleServiceAdapter struct {
	service module.ModuleService
}

func (a *moduleServiceAdapter) GetModuleByName(ctx context.Context, name string, userID primitive.ObjectID) (connectors.Module, error) {
	mod, err := a.service.GetModuleByName(ctx, name, userID)
	if err != nil {
		return connectors.Module{}, err
	}

	// Convert to connector Module type
	fields := make([]connectors.ModuleField, len(mod.Fields))
	for i, f := range mod.Fields {
		fields[i] = connectors.ModuleField{
			Name:     f.Name,
			Type:     string(f.Type),
			Label:    f.Label,
			Required: f.Required,
		}
	}

	return connectors.Module{
		Name:   mod.Name,
		Fields: fields,
	}, nil
}

func (s *DataSourceServiceImpl) CreateDataSource(ctx context.Context, ds *DataSource) error {
	ds.CreatedAt = time.Now()
	ds.UpdatedAt = time.Now()

	// Validate connection before creating
	if ds.IsActive {
		connector, err := s.createConnector(ctx, ds)
		if err != nil {
			return fmt.Errorf("failed to create connector: %w", err)
		}

		if err := connector.TestConnection(ctx); err != nil {
			return fmt.Errorf("connection test failed: %w", err)
		}
	}

	err := s.dataSourceRepo.Create(ctx, ds)
	if err == nil {
		s.auditService.LogChange(ctx, common_models.AuditActionCreate, "data_sources", ds.ID.Hex(), map[string]common_models.Change{
			"data_source": {New: ds},
		})
	}
	return err
}

func (s *DataSourceServiceImpl) GetDataSource(ctx context.Context, id string) (*DataSource, error) {
	return s.dataSourceRepo.Get(ctx, id)
}

func (s *DataSourceServiceImpl) ListDataSources(ctx context.Context) ([]DataSource, error) {
	return s.dataSourceRepo.List(ctx)
}

func (s *DataSourceServiceImpl) UpdateDataSource(ctx context.Context, id string, updates map[string]interface{}) error {
	oldDS, _ := s.GetDataSource(ctx, id)

	updates["updated_at"] = time.Now()
	err := s.dataSourceRepo.Update(ctx, id, updates)

	if err == nil {
		// Clear cached connector
		delete(s.connectors, id)

		s.auditService.LogChange(ctx, common_models.AuditActionUpdate, "data_sources", id, map[string]common_models.Change{
			"data_source": {Old: oldDS, New: updates},
		})
	}
	return err
}

func (s *DataSourceServiceImpl) DeleteDataSource(ctx context.Context, id string) error {
	oldDS, _ := s.GetDataSource(ctx, id)

	// Disconnect if cached
	if connector, exists := s.connectors[id]; exists {
		connector.Disconnect(ctx)
		delete(s.connectors, id)
	}

	err := s.dataSourceRepo.Delete(ctx, id)
	if err == nil {
		s.auditService.LogChange(ctx, common_models.AuditActionDelete, "data_sources", id, map[string]common_models.Change{
			"data_source": {Old: oldDS, New: "DELETED"},
		})
	}
	return err
}

func (s *DataSourceServiceImpl) TestDataSource(ctx context.Context, id string) error {
	ds, err := s.GetDataSource(ctx, id)
	if err != nil {
		return err
	}

	connector, err := s.getConnector(ctx, ds)
	if err != nil {
		return err
	}

	return connector.TestConnection(ctx)
}

func (s *DataSourceServiceImpl) QueryDataSource(ctx context.Context, req connectors.QueryRequest) (*connectors.QueryResponse, error) {
	// Get data source
	ds, err := s.GetDataSource(ctx, req.Source)
	if err != nil {
		return nil, fmt.Errorf("data source not found: %w", err)
	}

	if !ds.IsActive {
		return nil, fmt.Errorf("data source is not active")
	}

	// Get or create connector
	connector, err := s.getConnector(ctx, ds)
	if err != nil {
		return nil, err
	}

	// Execute query
	return connector.Query(ctx, req)
}

func (s *DataSourceServiceImpl) GetDataSourceSchema(ctx context.Context, sourceID string, module string) (*connectors.SchemaInfo, error) {
	ds, err := s.GetDataSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	connector, err := s.getConnector(ctx, ds)
	if err != nil {
		return nil, err
	}

	return connector.GetSchema(ctx, module)
}

func (s *DataSourceServiceImpl) QueryMultipleSources(ctx context.Context, queries []connectors.QueryRequest) (map[string]*connectors.QueryResponse, error) {
	results := make(map[string]*connectors.QueryResponse)

	for _, query := range queries {
		response, err := s.QueryDataSource(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to query source %s: %w", query.Source, err)
		}
		results[query.Source] = response
	}

	return results, nil
}

// getConnector gets or creates a connector for a data source
func (s *DataSourceServiceImpl) getConnector(ctx context.Context, ds *DataSource) (connectors.Connector, error) {
	// Check cache
	if conn, exists := s.connectors[ds.ID.Hex()]; exists {
		return conn, nil
	}

	// Create new connector
	connector, err := s.createConnector(ctx, ds)
	if err != nil {
		return nil, err
	}

	// Cache connector
	s.connectors[ds.ID.Hex()] = connector

	return connector, nil
}

// createConnector creates a new connector based on data source type
func (s *DataSourceServiceImpl) createConnector(ctx context.Context, ds *DataSource) (connectors.Connector, error) {
	var connector connectors.Connector

	switch ds.Type {
	case DataSourceTypeCRM, DataSourceTypeERP:
		connector = s.crmConnector
	case DataSourceTypePostgreSQL:
		connector = connectors.NewExternalDBConnector("postgresql")
		if err := connector.Connect(ctx, ds.Config); err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}
	case DataSourceTypeMySQL:
		connector = connectors.NewExternalDBConnector("mysql")
		if err := connector.Connect(ctx, ds.Config); err != nil {
			return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported data source type: %s", ds.Type)
	}

	return connector, nil
}
