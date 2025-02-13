package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
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
		solrURL := "http://solr:8983/solr"

		var params RelatedTermsParams
		if err := c.Bind(&params); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}

		// Build the request payload.
		reqBody := map[string]interface{}{
			"params": map[string]interface{}{
				"qf":         "text",
				"q":          params.Keyword,
				"fore":       "{!type=$defType qf=$qf v=$q}",
				"back":       "*:*",
				"defType":    "edismax",
				"rows":       0,
				"echoParams": "none",
				"omitHeader": "true",
			},
			"facet": map[string]interface{}{
				"body": map[string]interface{}{
					"type":     "terms",
					"field":    "text",
					"sort":     map[string]interface{}{"relatedness": "desc"},
					"mincount": 2,
					"limit":    8,
					"facet": map[string]interface{}{
						"relatedness": map[string]interface{}{
							"type": "func",
							"func": "relatedness($fore,$back)",
						},
					},
				},
			},
		}

		payload, err := json.Marshal(reqBody)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to marshal request"})
		}
		fmt.Println(string(payload))
		// Build Solr query URL.
		url := fmt.Sprintf("%s/%s/query", solrURL, params.Collection)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send post request"})
		}
		defer resp.Body.Close()

		var solrResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&solrResp); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to decode response"})
		}
		b, _ := json.Marshal(solrResp)
		fmt.Println(string(b))
		// Parse the response buckets.
		facets, ok := solrResp["facets"].(map[string]interface{})
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid facets in response"})
		}
		body, ok := facets["body"].(map[string]interface{})
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid body in facets"})
		}
		buckets, ok := body["buckets"].([]interface{})
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid buckets in response"})
		}

		result := make([]RelatedTerm, 0)
		for _, bucket := range buckets {
			bkt, ok := bucket.(map[string]interface{})
			if !ok {
				continue
			}
			var relatedVal, val string
			if rel, ok := bkt["relatedness"].(map[string]interface{}); ok {
				relatedVal = fmt.Sprintf("%v", rel["relatedness"])
			}
			val = fmt.Sprintf("%v", bkt["val"])
			// result.WriteString(fmt.Sprintf("%s\t%s\n", relatedVal, val))
			result = append(result, RelatedTerm{
				Term:        val,
				Relatedness: relatedVal,
			})
		}

		return c.JSON(http.StatusOK, result)
	}
}
