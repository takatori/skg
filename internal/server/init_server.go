package server

import (
	"github.com/labstack/echo/v4"
	"github.com/takatori/skg/internal"
	"github.com/takatori/skg/internal/server/handler"
)

func InitServer(config *internal.Config) (*echo.Echo, error) {
	e := echo.New()

	e.GET("/health", handler.NewHealthHandler())
	e.POST("/solr/setup", handler.NewSetupSolrHandler())
	e.POST("/solr/schema", handler.NewSetupSolrSchemaHandler())
	e.POST("/solr/feed", handler.NewFeedSolrDataHandler())
	e.POST("/skg/relatedTerms", handler.NewRelatedTermsHandler(config))

	return e, nil
}
