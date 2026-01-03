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

type UserClaims struct {
	UserID   string   `json:"user_id"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`            // Role Names
	Groups   []string `json:"groups,omitempty"` // User groups for ABAC
	RoleIDs  []string `json:"role_ids"`         // Role IDs
	jwt.RegisteredClaims
}

func GenerateToken(userID primitive.ObjectID, tenantID primitive.ObjectID, roleNames []string, roleIDs []string, groups []string) (string, error) {
	claims := UserClaims{
		UserID:   userID.Hex(),
		TenantID: tenantID.Hex(),
		Roles:    roleNames,
		RoleIDs:  roleIDs,
		Groups:   groups,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
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
