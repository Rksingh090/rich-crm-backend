package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var jwtSecret = []byte("secret")

// SetSecret allows injecting the secret from config
func SetSecret(secret string) {
	jwtSecret = []byte(secret)
}

// AppClaim represents role info per app
type AppClaim struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type UserClaims struct {
	UserID          string     `json:"user_id"`
	TenantID        string     `json:"tenant_id"`
	Roles           []string   `json:"roles"`            // Role Names
	Role            string     `json:"role"`             // Primary Role
	Apps            []AppClaim `json:"apps"`             // Enabled Apps with Roles
	Groups          []string   `json:"groups,omitempty"` // User groups for ABAC
	RoleIDs         []string   `json:"role_ids"`         // Role IDs
	IsPlatformAdmin bool       `json:"is_platform_admin"`
	jwt.RegisteredClaims
}

func GenerateToken(userID primitive.ObjectID, tenantID primitive.ObjectID, roleNames []string, roleIDs []string, groups []string, apps []AppClaim, isPlatformAdmin bool) (string, error) {
	primaryRole := ""
	if len(roleNames) > 0 {
		primaryRole = roleNames[0]
	}

	claims := UserClaims{
		UserID:          userID.Hex(),
		TenantID:        tenantID.Hex(),
		Roles:           roleNames,
		Role:            primaryRole,
		Apps:            apps,
		RoleIDs:         roleIDs,
		Groups:          groups,
		IsPlatformAdmin: isPlatformAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenSignatureInvalid
}
