package server

import (
	"github.com/labstack/echo/v4"
	"github.com/takatori/skg/internal"
	"github.com/takatori/skg/internal/infra"
	"github.com/takatori/skg/internal/server/handler"
)

func InitServer(config *internal.Config) (*echo.Echo, error) {
	e := echo.New()

	// Create a shared HTTP client
	httpClient := infra.NewHttpClient()

	// Create handlers with the shared HTTP client
	solrHandler := handler.NewSolrHandler(config, httpClient)
	relatedTermsHandler := handler.NewRelatedTermsHandlerWithClient(config, httpClient)

	// Register routes
	e.GET("/health", handler.NewHealthHandler())
	e.POST("/solr/setup", solrHandler.SetupSolrHandler())
	e.POST("/solr/schema", solrHandler.SetupSolrSchemaHandler())
	e.POST("/solr/feed", solrHandler.FeedSolrDataHandler())
	e.POST("/skg/relatedTerms", relatedTermsHandler.RelatedTermsEndpoint())

	return e, nil
}
