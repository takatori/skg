package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func NewHealthHandler() func(echo.Context) error {
	return func(c echo.Context) error {
		return c.String(
			http.StatusOK, "Hello, World! Skg",
		)
	}
}
