package middleware

import (
	"context"
	"net/http"
	"strings"

	"go-crm/pkg/utils"
)

// UserClaimsKey moved to pkg/utils

func AuthMiddleware(skipAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipAuth {
				// Inject dummy context for dev
				dummyClaims := &utils.UserClaims{
					UserID: "dev-admin-id",
					Roles:  []string{"val"}, // Matches nothing specific or can match admin if we hack it, but here just valid token struct
				}
				ctx := context.WithValue(r.Context(), utils.UserClaimsKey, dummyClaims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			claims, err := utils.ValidateToken(parts[1])
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), utils.UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
