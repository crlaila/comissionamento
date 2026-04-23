package middleware

import (
	"context"
	"net/http"

	"comissionamento/internal/handler"
	"comissionamento/internal/model"
	"comissionamento/internal/service"
)

type AuthMiddleware struct {
	authService *service.AuthService
}

func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// Authenticate validates JWT token and injects user context into request
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := handler.ExtractTokenFromHeader(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := m.authService.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Inject user info into request context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_email", claims.Email)
		ctx = context.WithValue(ctx, "user_role", claims.Role)
		ctx = context.WithValue(ctx, "user_name", claims.Name)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole ensures the user has one of the required roles
func (m *AuthMiddleware) RequireRole(requiredRoles ...model.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := r.Context().Value("user_role").(model.UserRole)
			if !ok {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Check if user's role is in required roles
			allowed := false
			for _, role := range requiredRoles {
				if userRole == role {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth validates JWT token if present, but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := handler.ExtractTokenFromHeader(r)
		if err == nil {
			if claims, err := m.authService.ValidateToken(token); err == nil {
				ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
				ctx = context.WithValue(ctx, "user_email", claims.Email)
				ctx = context.WithValue(ctx, "user_role", claims.Role)
				ctx = context.WithValue(ctx, "user_name", claims.Name)
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}
