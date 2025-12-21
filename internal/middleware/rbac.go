package middleware

import (
	"net/http"
	"slices"

	"go-crm/internal/service"
	"go-crm/pkg/utils"
)

// RequirePermission checks if the user has a specific permission
func RequirePermission(roleService service.RoleService, skipAuth bool, requiredPermission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if skipAuth {
			next.ServeHTTP(w, r)
			return
		}

		claims, ok := r.Context().Value(UserClaimsKey).(*utils.UserClaims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check permissions via service
		permissions, err := roleService.GetPermissionsForRoles(r.Context(), claims.Roles)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !slices.Contains(permissions, requiredPermission) {
			http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}
