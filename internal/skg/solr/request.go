package solr

import (
	"fmt"

	"github.com/takatori/skg/internal/skg"
)

// transformRequest generates a faceted Solr SKG request from one or more multi-nodes.
// Each multi-node can be either a single Node or a slice of Node.
// Subsequent nodes are nested as facets of their parent nodes.
func transformRequest(multiNodes [][]skg.Query) map[string]interface{} {

	request := generateRequestRoot()
	params := request["params"].(map[string]interface{})
	// Start with the root as the only parent node.
	parentNodes := []map[string]interface{}{request}

	for i, nodes := range multiNodes {
		var currentFacets []map[string]interface{}

		for j, node := range nodes {
			if node.Name == "" {
				node.Name = defaultNodeName(i, j)
			}
			facets := generateFacets(node.Name, node.Values, node.Field, node.MinOccurrence, node.Limit, node.MinPopularity, node.DefaultOperator)
			currentFacets = append(currentFacets, facets...)

			// Attach the generated facets to each parent node.
			for _, parentNode := range parentNodes {
				facetField, ok := parentNode["facet"].(map[string]interface{})
				if !ok {
					facetField = map[string]interface{}{}
					parentNode["facet"] = facetField
				}
				for k, facet := range facets {
					key := fmt.Sprintf("%s_%d", node.Name, k)
					parentNode["facet"].(map[string]interface{})[key] = facet
				}
			}
			// If the node has a values array, add query parameters to the root.
			if len(node.Values) > 0 {
				for k, value := range node.Values {
					key := fmt.Sprintf("%s_%d_query", node.Name, k)
					params[key] = value
				}
			}
		}
		// Update parentNodes to be the facets from this multi-node.
		parentNodes = currentFacets
	}
	return request
}

// generateFacets returns a slice of facet definitions based on the provided node parameters.
func generateFacets(name string, values []string, field string, minOccurrence *int, limit *int, minPopularity *int, defaultOperator string) []map[string]interface{} {
	// Choose facet type based on values.
	facetType := "terms"
	if len(values) > 0 {
		facetType = "query"
	}

	// Set limit default to 10 if not provided.
	facLimit := 10
	if limit != nil {
		facLimit = *limit
	}

	baseFacet := map[string]interface{}{
		"type":  facetType,
		"limit": facLimit,
		"sort": map[string]interface{}{
			"relatedness": "desc",
		},
		"facet": map[string]interface{}{
			"relatedness": map[string]interface{}{
				"type": "func",
				"func": "relatedness($fore,$back)",
			},
		},
	}
	if minOccurrence != nil {
		baseFacet["mincount"] = *minOccurrence
	}
	if minPopularity != nil {
		// Add to nested facet.
		if facetMap, ok := baseFacet["facet"].(map[string]interface{}); ok {
			if rel, ok := facetMap["relatedness"].(map[string]interface{}); ok {
				rel["min_popularity"] = *minPopularity
			}
		}
	}
	if field != "" {
		baseFacet["field"] = field
	}

	var facets []map[string]interface{}
	if len(values) > 0 {
		// When values are present, remove mincount (if any) as per the Python code.
		if minOccurrence != nil {
			delete(baseFacet, "mincount")
		}
		if limit == nil {
			delete(baseFacet, "limit")
		}
		// For each value, create a facet copy with the appropriate query.
		for i := range values {
			facetCopy := deepCopyMap(baseFacet)
			queryStr := fmt.Sprintf("{!edismax q.op=%s qf=%s v=$%s_%d_query}", getDefaultOperator(defaultOperator), field, name, i)
			facetCopy["query"] = queryStr
			facets = append(facets, facetCopy)
		}
	} else {
		facets = append(facets, baseFacet)
	}
	return facets
}

// generateRequestRoot creates the basic request structure.
func generateRequestRoot() map[string]interface{} {
	return map[string]interface{}{
		"limit": 0,
		"params": map[string]interface{}{
			"q":       "*:*",
			"fore":    "{!${defType} v=$q}",
			"back":    "*:*",
			"defType": "edismax",
		},
		"facet": map[string]interface{}{},
	}
}

// getDefaultOperator returns the operator or "AND" if empty.
func getDefaultOperator(op string) string {
	if op == "" {
		return "AND"
	}
	return op
}

// deepCopyMap returns a deep copy of a map.
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			newMap[k] = deepCopyMap(val)
		default:
			newMap[k] = v
		}
	}
	return newMap
}

// defaultNodeName generates a default name based on indices.
func defaultNodeName(i, j int) string {
	if j == 0 {
		return fmt.Sprintf("f%d", i)
	}
	return fmt.Sprintf("f%d_%d", i, j)
}
