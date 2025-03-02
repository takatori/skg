package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"github.com/takatori/skg/internal/skg"
	"github.com/takatori/skg/internal/skg/solr"
)

func NewSkgHandler() func(echo.Context) error {
	return func(c echo.Context) error {
		return c.String(
			http.StatusOK, "Hello, World! Skg",
		)
	}
}

type RelatedTermsParams struct {
	Keyword    string `json:"keyword" validate:"required"`
	Collection string `json:"collection" validate:"required"`
}

type RelatedTerm struct {
	Term        string `json:"term"`
	Relatedness string `json:"relatedness"`
}

// NewRelatedTermsHandler queries Solr and returns formatted related terms.
func NewRelatedTermsHandler() func(echo.Context) error {
	return func(c echo.Context) error {
		// Define the Solr URL and collection.

		var params RelatedTermsParams
		if err := c.Bind(&params); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}

		queries := [][]skg.Query{
			{
				{
					Field: "text",
					Values: []string{
						params.Keyword,
					},
				},
			}, {
				{
					Field:         "text",
					MinOccurrence: lo.ToPtr(2),
					Limit:         lo.ToPtr(8),
				},
			},
		}

		skg := solr.NewSolrSemanticKnowledgeGraph()

		result, err := skg.Traverse(queries)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, result)
	}
}
