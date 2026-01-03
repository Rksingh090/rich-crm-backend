package connectors

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// ExternalDBConnector connects to external SQL databases
type ExternalDBConnector struct {
	dbType string // "postgresql" or "mysql"
	db     *sql.DB
}

// NewExternalDBConnector creates a new external database connector
func NewExternalDBConnector(dbType string) Connector {
	return &ExternalDBConnector{
		dbType: dbType,
	}
}

// Connect establishes connection to external database
func (c *ExternalDBConnector) Connect(ctx context.Context, config map[string]interface{}) error {
	connStr, err := c.buildConnectionString(config)
	if err != nil {
		return fmt.Errorf("failed to build connection string: %w", err)
	}

	driver := c.dbType
	if c.dbType == "postgresql" {
		driver = "postgres"
	}

	db, err := sql.Open(driver, connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	c.db = db
	return nil
}

// Disconnect closes the database connection
func (c *ExternalDBConnector) Disconnect(ctx context.Context) error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Query executes a query against the external database
func (c *ExternalDBConnector) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	query, args := c.buildSQLQuery(req)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	data, err := c.rowsToMaps(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to process query results: %w", err)
	}

	return &QueryResponse{
		Data:       data,
		TotalCount: int64(len(data)),
	}, nil
}

// GetSchema returns schema information for a table
func (c *ExternalDBConnector) GetSchema(ctx context.Context, module string) (*SchemaInfo, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	var query string
	if c.dbType == "postgresql" {
		query = `
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = $1
			ORDER BY ordinal_position
		`
	} else { // mysql
		query = `
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = ?
			ORDER BY ordinal_position
		`
	}

	rows, err := c.db.QueryContext(ctx, query, module)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	defer rows.Close()

	schema := &SchemaInfo{
		Module: module,
		Fields: []FieldInfo{},
	}

	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault sql.NullString

		if err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}

		schema.Fields = append(schema.Fields, FieldInfo{
			Name:       columnName,
			Type:       dataType,
			Label:      columnName,
			IsRequired: isNullable == "NO",
		})
	}

	return schema, nil
}

// TestConnection tests if the database connection is valid
func (c *ExternalDBConnector) TestConnection(ctx context.Context) error {
	if c.db == nil {
		return fmt.Errorf("database connection not established")
	}
	return c.db.PingContext(ctx)
}

// GetType returns the connector type
func (c *ExternalDBConnector) GetType() string {
	return c.dbType
}

// buildConnectionString creates a connection string from config
func (c *ExternalDBConnector) buildConnectionString(config map[string]interface{}) (string, error) {
	host, _ := config["host"].(string)
	port, _ := config["port"].(float64)
	database, _ := config["database"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)

	if host == "" || database == "" || username == "" {
		return "", fmt.Errorf("missing required connection parameters")
	}

	if port == 0 {
		if c.dbType == "postgresql" {
			port = 5432
		} else {
			port = 3306
		}
	}

	if c.dbType == "postgresql" {
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			host, int(port), username, password, database,
		), nil
	}

	// MySQL
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true",
		username, password, host, int(port), database,
	), nil
}

// buildSQLQuery constructs a SQL query from QueryRequest
func (c *ExternalDBConnector) buildSQLQuery(req QueryRequest) (string, []interface{}) {
	var query strings.Builder
	var args []interface{}
	argIndex := 1

	// SELECT clause
	query.WriteString("SELECT ")
	if len(req.Fields) > 0 {
		query.WriteString(strings.Join(req.Fields, ", "))
	} else {
		query.WriteString("*")
	}

	// FROM clause
	query.WriteString(fmt.Sprintf(" FROM %s", req.Module))

	// WHERE clause
	if len(req.Filters) > 0 {
		query.WriteString(" WHERE ")
		conditions := []string{}
		for field, value := range req.Filters {
			placeholder := c.getPlaceholder(argIndex)
			conditions = append(conditions, fmt.Sprintf("%s = %s", field, placeholder))
			args = append(args, value)
			argIndex++
		}
		query.WriteString(strings.Join(conditions, " AND "))
	}

	// GROUP BY clause (for aggregation)
	if req.Aggregation != nil && len(req.Aggregation.GroupBy) > 0 {
		query.WriteString(" GROUP BY ")
		query.WriteString(strings.Join(req.Aggregation.GroupBy, ", "))
	}

	// ORDER BY clause
	if len(req.Sort) > 0 {
		query.WriteString(" ORDER BY ")
		sortClauses := []string{}
		for field, direction := range req.Sort {
			dir := "ASC"
			if direction == -1 {
				dir = "DESC"
			}
			sortClauses = append(sortClauses, fmt.Sprintf("%s %s", field, dir))
		}
		query.WriteString(strings.Join(sortClauses, ", "))
	}

	// LIMIT and OFFSET
	if req.Limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", req.Limit))
	}
	if req.Offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", req.Offset))
	}

	return query.String(), args
}

// getPlaceholder returns the appropriate placeholder for the database type
func (c *ExternalDBConnector) getPlaceholder(index int) string {
	if c.dbType == "postgresql" {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

// rowsToMaps converts SQL rows to a slice of maps
func (c *ExternalDBConnector) rowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := []map[string]interface{}{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		result = append(result, row)
	}

	return result, nil
}
