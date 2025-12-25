package middleware

import (
	"array-assessment/internal/dto"
	"array-assessment/internal/errors"
	"array-assessment/internal/handlers"
	"array-assessment/internal/services"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/labstack/echo/v4"
)

func RequireAuthAccount(northWindService services.NorthWindServiceInterface) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			bodyBytes, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return handlers.SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
			}

			c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var req dto.CreateAccountRequest
			if err := json.Unmarshal(bodyBytes, &req); err != nil {
				return handlers.SendError(c, errors.ValidationGeneral, errors.WithDetails("Invalid request body"))
			}

			fmt.Println(req)

			res, err := northWindService.AuthAccount(context.TODO(), dto.NorthWindAccountRequestDto{
				AccountHolderName: req.AccountHolderName,
				AccountNumber:     req.AccountNumber,
				RoutingNumber:     req.RoutingNumber,
			})
			if err != nil {
				return handlers.SendError(c, errors.NorthWindAccountError, errors.WithDetails("Error while authenticating northwind account"))
			}

			if !res.AccountExists {
				return handlers.SendError(c, errors.NorthWindAccountNotFound, errors.WithDetails("Northwind account not found"))
			}

			c.Set("initialDeposit", res.AvailableBalance)
			c.Set("accountNumber", req.AccountNumber)
			c.Set("routingNumber", req.RoutingNumber)
			c.Set("accountHolderName", req.AccountHolderName)

			return next(c)
		}
	}
}
