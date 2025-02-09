package server

import (
	"github.com/labstack/echo/v4"
	"github.com/takatori/skg/internal/server/handler"
)

func InitServer(e *echo.Echo) {

	e.GET("/health", handler.NewHealthHandler())
	e.POST("/solr/setup", handler.NewSetupSolrHandler())
}
