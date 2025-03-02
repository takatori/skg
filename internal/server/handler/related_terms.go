package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"github.com/takatori/skg/internal/skg"
	"github.com/takatori/skg/internal/skg/solr"
)

// RelatedTermsParams defines the parameters for the related terms API
type RelatedTermsParams struct {
	Keyword    string `json:"keyword" validate:"required"`
	Collection string `json:"collection" validate:"required"`
}

// RelatedTerm represents a single term related to the input keyword
type RelatedTerm struct {
	Term        string  `json:"term"`
	Relatedness float64 `json:"relatedness"`
}

// NewRelatedTermsHandler creates a handler that queries Solr and returns formatted related terms.
func NewRelatedTermsHandler() func(echo.Context) error {
	return func(c echo.Context) error {
		// Parse and validate request parameters
		params, err := parseParams(c)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		// Build queries for the semantic knowledge graph
		queries := buildQueries(params.Keyword)

		// Query the semantic knowledge graph
		skgInstance := solr.NewSolrSemanticKnowledgeGraph()
		result, err := skgInstance.Traverse(queries)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Process results into related terms
		relatedTerms := extractRelatedTerms(result)

		return c.JSON(http.StatusOK, relatedTerms)
	}
}

// parseParams extracts and validates the request parameters
func parseParams(c echo.Context) (RelatedTermsParams, error) {
	var params RelatedTermsParams
	if err := c.Bind(&params); err != nil {
		return params, err
	}
	return params, nil
}

// buildQueries constructs the query structure for the semantic knowledge graph
func buildQueries(keyword string) [][]skg.Query {
	return [][]skg.Query{
		{
			{
				Field: "text",
				Values: []string{
					keyword,
				},
			},
		},
		{
			{
				Field:         "text",
				MinOccurrence: lo.ToPtr(2),
				Limit:         lo.ToPtr(8),
			},
		},
	}
}

// extractRelatedTerms processes the SKG result into a list of related terms
func extractRelatedTerms(result map[string]skg.Traversal) []RelatedTerm {
	var relatedTerms []RelatedTerm

	for _, item := range result {
		// Skip if there are no values
		if len(item.Values) == 0 {
			continue
		}

		// Process each traversal in the first value
		for _, traversal := range item.Values[0].Traversals {
			for _, value := range traversal.Values {
				relatedTerms = append(relatedTerms, RelatedTerm{
					Term:        value.Key,
					Relatedness: value.Relatedness,
				})
			}
		}
	}

	return relatedTerms
}
