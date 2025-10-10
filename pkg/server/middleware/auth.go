package middleware

import (
	"TrackMe/internal/domain/user"
	"TrackMe/pkg/jwt"
	"TrackMe/pkg/server/response"
	"context"
	"errors"
	"net/http"
	"strings"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// AuthMiddleware validates JWT token and adds user info to context
func AuthMiddleware(tokenManager *jwt.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, r, errors.New("authorization header required"))
				return
			}

			// Extract token - support both "Bearer <token>" and "<token>" formats
			var token string
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				// Format: "Bearer <token>"
				token = parts[1]
			} else if len(parts) == 1 {
				// Format: "<token>" (without Bearer prefix)
				token = parts[0]
			} else {
				response.Unauthorized(w, r, errors.New("invalid authorization header format"))
				return
			}

			claims, err := tokenManager.ValidateToken(token)
			if err != nil {
				response.Unauthorized(w, r, err)
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks if user has required role
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*jwt.Claims)
			if !ok {
				response.Forbidden(w, r, errors.New("user not found in context"))
				return
			}

			// Check if user has required role
			hasRole := false
			for _, role := range roles {
				if claims.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				response.Forbidden(w, r, errors.New("insufficient permissions"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireSuperUserOrAdmin allows only super_user or admin
func RequireSuperUserOrAdmin() func(http.Handler) http.Handler {
	return RequireRole(user.RoleSuperUser, user.RoleAdmin)
}

// RequireSuperUser allows only super_user
func RequireSuperUser() func(http.Handler) http.Handler {
	return RequireRole(user.RoleSuperUser)
}

// RequireAdminOrManager allows admin or manager (read-only for manager)
func RequireAdminOrManager() func(http.Handler) http.Handler {
	return RequireRole(user.RoleAdmin, user.RoleManager)
}

// GetUserFromContext retrieves user claims from context
func GetUserFromContext(ctx context.Context) (*jwt.Claims, error) {
	claims, ok := ctx.Value(UserContextKey).(*jwt.Claims)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return claims, nil
}
