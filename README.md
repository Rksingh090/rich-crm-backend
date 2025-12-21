# Go CRM API

A robust, scalable CRM backend built with Go, MongoDB, and modern practices. features dynamic module schema creation, RBAC, and a service-oriented architecture.

## 🚀 Features

- **Generic Repository Pattern**: Decoupled data access layer.
- **Service Layer**: Business logic separation.
- **Dynamic Module Builder**: Create custom CRM modules (Leads, Deals, etc.) with flexible schemas via API.
- **Generic Data Management**: Insert, Update, and Delete records with automatic validation against module schemas.
- **Full CRUD Operations**: Complete API support for managing both Module definitions and Data records.
- **Comprehensive Field Types**: Support for Text, Number, Date, Email, Phone, File, Lookup, Select, and more.
- **Role-Based Access Control (RBAC)**: Granular permission management.
- **Authentication**: JWT-based auth with secure password hashing.
- **Hot Reloading**: Integrated with `air` for rapid development.
- **API Documentation**: Auto-generated Swagger UI.
- **Development Mode**: Optional global auth bypass for easy testing.

## 🛠 Prerequisites

- **Go**: 1.22 or later
- **MongoDB**: Running instance (local or cloud)
- **Make**: For running build commands
- **Air**: For hot reloading (`go install github.com/air-verse/air@latest`)
- **Swag**: For documentation (`go install github.com/swaggo/swag/cmd/swag@latest`)

## ⚙️ Setup & Configuration

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/yourusername/go-crm.git
    cd go-crm
    ```

2.  **Environment Variables**:
    Copy `.env.example` to `.env` and configure your settings.
    ```bash
    cp .env.example .env
    ```

    **Key Variables**:
    - `PORT`: Server port (default: 8000)
    - `MONGO_URI`: MongoDB connection string
    - `DB_NAME`: Database name
    - `JWT_SECRET`: Secret key for token signing
    - `SKIP_AUTH`: Set to `true` to bypass Auth/RBAC (Development only!)

## 🏃‍♂️ Running the Project

### Development Mode (Hot Reload)
Use `make run` to start the server with hot reloading. It will strictly regenerate Swagger docs on every change.
```bash
make run
```

### Build & Run Binary
```bash
go build -o bin/api cmd/api/main.go
./bin/api
```

### Generate Documentation
Manually regenerate Swagger docs:
```bash
make swagger
```

## 📚 API Documentation

Once the server is running, access the interactive Swagger UI at:
**[http://localhost:8000/swagger/index.html](http://localhost:8000/swagger/index.html)**

### Key Endpoints

#### Modules (`/modules`)
- `POST /modules`: Create a new module definition (e.g., "Leads").
- `GET /modules`: List all modules.
- `GET /modules/{name}`: Get module schema.
- `PUT /modules/{name}`: Update module schema.
- `DELETE /modules/{name}`: Delete a module.

#### Records (`/modules/{name}/records`)
- `POST /modules/{name}/records`: Insert data (validated against schema).
- `PUT /modules/{name}/records/{id}`: Update data (partial updates supported).
- `DELETE /modules/{name}/records/{id}`: Delete data.

#### Auth & Admin
- `POST /register`, `POST /login`: User authentication.
- `GET /admin`: RBAC protected route example.

## 🏗 Project Structure

```
├── cmd/api/            # Application entry point
├── docs/               # Generated Swagger files
├── internal/
│   ├── config/         # Configuration loader
│   ├── database/       # DB connection logic
│   ├── handlers/       # HTTP transport layer
│   ├── middleware/     # Auth and RBAC middleware
│   ├── models/         # Domain data definitions
│   ├── repository/     # Data access interfaces & implementation
│   ├── routes/         # Route definitions and wiring
│   └── service/        # Business logic
└── pkg/utils/          # Shared utilities (JWT, etc.)
```

## 🧪 Testing

To run unit tests (future implementation):
```bash
go test ./...
```
