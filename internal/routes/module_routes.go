package routes

import (
	"go-crm/internal/handlers"
	"go-crm/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func RegisterModuleRoutes(r chi.Router, moduleHandler *handlers.ModuleHandler, recordHandler *handlers.RecordHandler, skipAuth bool) {
	r.Route("/modules", func(r chi.Router) {
		// Apply auth middleware to all module routes
		r.Use(middleware.AuthMiddleware(skipAuth))

		// Module CRUD
		r.Post("/", moduleHandler.CreateModule)
		r.Get("/", moduleHandler.ListModules)

		// Specific module operations
		r.Route("/{name}", func(r chi.Router) {
			r.Get("/", moduleHandler.GetModule)
			r.Put("/", moduleHandler.UpdateModule)
			r.Delete("/", moduleHandler.DeleteModule)

			// Records for this module
			r.Route("/records", func(r chi.Router) {
				r.Get("/", recordHandler.ListRecords)
				r.Post("/", recordHandler.CreateRecord)

				// Specific record operations
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", recordHandler.GetRecord)
					r.Put("/", recordHandler.UpdateRecord)
					r.Delete("/", recordHandler.DeleteRecord)
				})
			})
		})
	})
}
