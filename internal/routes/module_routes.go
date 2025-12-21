package routes

import (
	"net/http"
	"strings"

	"go-crm/internal/handlers"
	"go-crm/internal/middleware"
)

func RegisterModuleRoutes(mux *http.ServeMux, moduleHandler *handlers.ModuleHandler, recordHandler *handlers.RecordHandler, skipAuth bool) {
	// Apply Auth Middleware to all module routes
	// or specific ones. For now, let's protect them.

	createHandler := http.HandlerFunc(moduleHandler.CreateModule)
	listHandler := http.HandlerFunc(moduleHandler.ListModules)

	// Create Module (Protected)
	mux.Handle("/modules", middleware.AuthMiddleware(skipAuth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createHandler.ServeHTTP(w, r)
		case http.MethodGet:
			listHandler.ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Create Record in Module (Protected)
	// POST /modules/{name}/records
	// We need to parse path manually if we use /modules/ prefix match in Std Mux
	// Or we can register a specific pattern if we know it? No, dynamic.
	// We'll hook into /modules/ and check 4th part of path?
	// /modules/{name}/records -> parts: ["", "modules", "name", "records"]

	// Let's make a unified handler for /modules/ prefix
	unifiedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		// paths:
		// GET  /modules/{name}
		// PUT  /modules/{name}
		// DELETE /modules/{name}
		// POST /modules/{name}/records
		// GET  /modules/{name}/records  <-- NEW
		// Check if it's a specific record operation
		// Path: /modules/{name}/records/{id}
		parts := strings.Split(r.URL.Path, "/")
		// parts: ["", "modules", "{name}", "records", "{id}"] -> len 5
		if len(parts) == 5 && parts[2] != "" && parts[3] == "records" && parts[4] != "" {
			r.SetPathValue("id", parts[4])
			switch r.Method {
			case http.MethodGet:
				recordHandler.GetRecord(w, r)
			case http.MethodPut:
				recordHandler.UpdateRecord(w, r)
			case http.MethodDelete:
				recordHandler.DeleteRecord(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// List or Create
		// Path: /modules/{name}/records
		if len(parts) == 4 && parts[2] != "" && parts[3] == "records" {
			switch r.Method {
			case http.MethodGet:
				recordHandler.ListRecords(w, r)
			case http.MethodPost:
				recordHandler.CreateRecord(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		if len(pathParts) == 2 {
			// /modules/{name}
			switch r.Method {
			case http.MethodGet:
				moduleHandler.GetModule(w, r)
			case http.MethodPut:
				moduleHandler.UpdateModule(w, r)
			case http.MethodDelete:
				moduleHandler.DeleteModule(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		http.NotFound(w, r)
	})

	mux.Handle("/modules/", middleware.AuthMiddleware(skipAuth)(unifiedHandler))
}
