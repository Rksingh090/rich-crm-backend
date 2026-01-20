# Backend API Features

This document outlines the comprehensive feature set of the Go-CRM Backend API, designed to support both CRM and ERP functionalities.

## Core Services

### 1. Authentication & Identity
- **Endpoints**: `/api/auth/*`
- **Description**: Handles user registration, login, logout, and password management. Supports JWT-based authentication for secure access.
- **Key Features**:
    - Login/Register with Email/Password.
    - Password Reset flows.
    - Session management.

### 2. User & Role Management
- **Endpoints**: `/api/users`, `/api/roles`, `/api/groups`
- **Description**: Managing the workforce hierarchy and access controls.
- **Key Features**:
    - **Users**: Create, update, and manage system users.
    - **Roles**: Define roles (Admin, Manager, Sales Rep) with specific permissions.
    - **Groups**: Group users for shared access and assignment (e.g., "Sales Team").

### 3. Organization (Multi-Tenancy)
- **Endpoints**: `/api/organizations`
- **Description**: Supports multi-tenancy where each organization operates in isolation.
- **Key Features**:
    - Create and manage Organizations (Tenants).
    - Application Context switching (`X-Rich-App` header for CRM/ERP).

### 4. Permissions & Security
- **Endpoints**: `/api/permissions`
- **Description**: Granular control over what users can see and do.
- **Key Features**:
    - Module-level permissions (Create, Read, Update, Delete).
    - Field-level security (Hide specific fields like "Salary").
    - Profile-based access control.

---

## Data Engine

### 5. Module System (Dynamic Schema)
- **Endpoints**: `/api/modules`
- **Description**: The heart of the system. Allows defining dynamic data structures (schemas) without code changes.
- **Key Features**:
    - **CRUD Modules**: Create custom modules (e.g., "Projects", "Vehicles").
    - **Split Definitions**: Separate `crm_modules.json` and `erp_modules.json` for domain-specific schemas.
    - **Field Types**: Supports Text, Number, Date, Select, Lookup, File, Subforms, and more.

### 6. Record Management
- **Endpoints**: `/api/records/{moduleName}`
- **Description**: Universal CRUD interface for all module data.
- **Key Features**:
    - Create, Read, Update, Delete records for ANY module.
    - **Complex Filtering**: Filter by any field, operator (eq, contains, gt, lt).
    - **Relationships**: Handle Lookups and Related Lists automatically.

### 7. Global Search
- **Endpoints**: `/api/search`
- **Description**: Full-text search across all modules.
- **Key Features**:
    - unified search bar functionality.
    - Index-based retrieval.

### 8. Bulk Operations
- **Endpoints**: `/api/bulk/operations`
- **Description**: Perform actions on large datasets efficiently.
- **Key Features**:
    - Bulk Delete.
    - Bulk Update (mass edit).
    - Bulk Duplicate.

### 9. File Storage
- **Endpoints**: `/api/files`
- **Description**: Manage attachments and documents.
- **Key Features**:
    - Upload/Download files.
    - Link files to specific records.

---

## Business Logic Features

### 10. Activity Timeline
- **Endpoints**: `/api/activities`, `/api/activities/timeline`
- **Description**: Tracks interactions with customers.
- **Key Features**:
    - **Unified Timeline**: Aggregates Calls, Meetings, and Tasks into a single chronological view.
    - Calendar integration logic.

### 11. Approvals
- **Endpoints**: `/api/approvals`
- **Description**: Process governance.
- **Key Features**:
    - Define approval process for records (e.g., Discount Approval).
    - Approve/Reject actions with history.

### 12. Automation
- **Endpoints**: `/api/automation/rules`
- **Description**: Automate repetitive tasks.
- **Key Features**:
    - Trigger actions (Email, Task Creation) based on record changes.
    - Workflow Rules engine.

### 13. Ticketing System
- **Endpoints**: `/api/tickets`
- **Description**: Customer support and issue tracking.
- **Key Features**:
    - Ticket lifecycle management.
    - SLA tracking and escalation.

### 14. Inventory (ERP)
- **Endpoints**: `/api/inventory`
- **Description**: Product and stock management.
- **Key Features**:
    - Product definitions (Prices, SKU).
    - Stock level tracking.

---

## Analytics & Reporting

### 15. Reports & Charts
- **Endpoints**: `/api/reports`, `/api/charts`
- **Description**: Turn data into insights.
- **Key Features**:
    - Create tabular or summary reports.
    - Generate visual charts (Bar, Pie, Line).
    - Export data to Excel/CSV.

### 16. Dashboards
- **Endpoints**: `/api/dashboards`
- **Description**: Personalized layout of charts and metrics.
- **Key Features**:
    - Drag-and-drop widget arrangement.
    - KPI monitoring.

### 17. Audit Logs
- **Endpoints**: `/api/audit`
- **Description**: Security and compliance tracking.
- **Key Features**:
    - Record every change (Who, What, When).
    - "Old Value" vs "New Value" comparison (Diff).

---

## System Tools

### 18. Webhooks
- **Endpoints**: `/api/webhooks`
- **Description**: Integration with external systems.
- **Key Features**:
    - Trigger HTTP callbacks on system events.

### 19. Cron Service
- **Endpoints**: Internal Service
- **Description**: Background task scheduling.
- **Key Features**:
    - Scheduled jobs (e.g., Midnight sync, SLA checks).

### 20. Notification Center
- **Endpoints**: `/api/notifications`
- **Description**: User alerts.
- **Key Features**:
    - In-app alerts for assigned tasks, mentions, or approvals.

### 21. Email Templates
- **Endpoints**: `/api/email-templates`
- **Description**: Standardized communication.
- **Key Features**:
    - Create HTML templates with placeholders (Merge fields).
    - Send emails using these templates.