package middleware

import (
	"array-assessment/internal/errors"
	"array-assessment/internal/handlers"
	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RequireAuth creates a middleware that requires a valid JWT token
// and checks that the token has not been blacklisted (e.g., after logout)
func RequireAuth(tokenService services.TokenServiceInterface, blacklistedTokenRepo repositories.BlacklistedTokenRepositoryInterface) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return handlers.SendError(c, errors.AuthMissingToken)
			}

			token, err := tokenService.ExtractTokenFromHeader(authHeader)
			if err != nil {
				return handlers.SendError(c, errors.AuthInvalidTokenFormat)
			}

			claims, err := tokenService.ValidateAccessToken(token)
			if err != nil {
				if err == services.ErrExpiredToken {
					return handlers.SendError(c, errors.AuthExpiredToken)
				}
				return handlers.SendError(c, errors.AuthInvalidTokenFormat)
			}

			blacklistedToken, err := blacklistedTokenRepo.GetByJTI(claims.ID)
			if err == nil && blacklistedToken != nil {
				return handlers.SendError(c, errors.AuthInvalidTokenFormat, errors.WithDetails("Token has been revoked"))
			}

			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				return handlers.SendError(c, errors.AuthInvalidTokenFormat, errors.WithDetails("Invalid user ID in token"))
			}

			c.Set("user_id", userID)
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
			c.Set("token_jti", claims.ID)
			c.Set("is_admin", claims.Role == models.RoleAdmin)

			user := map[string]interface{}{
				"id":    userID,
				"email": claims.Email,
				"role":  claims.Role,
			}
			c.Set("user", user)

			return next(c)
		}
	}
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(requiredRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole, ok := c.Get("user_role").(string)
			if !ok {
				return handlers.SendError(c, errors.AuthInvalidTokenFormat, errors.WithDetails("User role not found in token"))
			}

			for _, role := range requiredRoles {
				if userRole == role {
					return next(c)
				}
			}

			return handlers.SendError(c, errors.AuthInsufficientPermission)
		}
	}
}

// RequireAdmin is a convenience middleware that requires admin role
func RequireAdmin() echo.MiddlewareFunc {
	return RequireRole(models.RoleAdmin)
}
