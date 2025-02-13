package server

import (
	"github.com/labstack/echo/v4"
	"github.com/takatori/skg/internal/server/handler"
)

func InitServer(e *echo.Echo) {

	e.GET("/health", handler.NewHealthHandler())
	e.POST("/solr/setup", handler.NewSetupSolrHandler())
	e.POST("/solr/schema", handler.NewSetupSolrSchemaHandler())
	e.POST("/solr/feed", handler.NewFeedSolrDataHandler())
	e.POST("/skg/relatedTerms", handler.NewRelatedTermsHandler())
}
