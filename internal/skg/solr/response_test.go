package solr

import (
	"encoding/json"
	"testing"

	"github.com/takatori/skg/internal/skg"
)

func TestRemoveSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"f0_0", "f0"},
		{"f1_0", "f1"},
		{"noSuffix", "noSuffix"},
		{"multiple_underscore_test_1", "multiple_underscore_test"},
		{"", ""},
	}

	for _, test := range tests {
		result := removeSuffix(test.input)
		if result != test.expected {
			t.Errorf("removeSuffix(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestExtractRelatedness(t *testing.T) {
	tests := []struct {
		name     string
		node     map[string]interface{}
		expected float64
	}{
		{
			name: "valid relatedness",
			node: map[string]interface{}{
				"count": float64(100),
				"relatedness": map[string]interface{}{
					"relatedness": float64(0.75),
				},
			},
			expected: 0.75,
		},
		{
			name: "no count",
			node: map[string]interface{}{
				"relatedness": map[string]interface{}{
					"relatedness": float64(0.75),
				},
			},
			expected: 0.0,
		},
		{
			name: "count zero",
			node: map[string]interface{}{
				"count": float64(0),
				"relatedness": map[string]interface{}{
					"relatedness": float64(0.75),
				},
			},
			expected: 0.0,
		},
		{
			name: "no relatedness map",
			node: map[string]interface{}{
				"count": float64(100),
			},
			expected: 0.0,
		},
		{
			name: "no relatedness value",
			node: map[string]interface{}{
				"count":       float64(100),
				"relatedness": map[string]interface{}{},
			},
			expected: 0.0,
		},
		{
			name: "relatedness not float",
			node: map[string]interface{}{
				"count": float64(100),
				"relatedness": map[string]interface{}{
					"relatedness": "not a float",
				},
			},
			expected: 0.0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := extractRelatedness(test.node)
			if result != test.expected {
				t.Errorf("extractRelatedness() = %v, expected %v", result, test.expected)
			}
		})
	}
}

func TestTransformNode(t *testing.T) {
	// Simple node with no nested traversals
	simpleNode := map[string]interface{}{
		"val":   "test",
		"count": float64(100),
		"relatedness": map[string]interface{}{
			"relatedness": float64(0.75),
		},
	}

	// Node with nested traversals
	nestedNode := map[string]interface{}{
		"val":   "parent",
		"count": float64(100),
		"relatedness": map[string]interface{}{
			"relatedness": float64(0.75),
		},
		"f1_0": map[string]interface{}{
			"buckets": []interface{}{
				map[string]interface{}{
					"val":   "child1",
					"count": float64(50),
					"relatedness": map[string]interface{}{
						"relatedness": float64(0.5),
					},
				},
			},
		},
	}

	t.Run("simple node", func(t *testing.T) {
		result := transformNode(simpleNode, nil)
		expected := skg.Node{
			Key:         "test",
			Relatedness: 0.75,
			Traversals:  []skg.Traversal{},
		}

		if result.Key != expected.Key || result.Relatedness != expected.Relatedness {
			t.Errorf("transformNode() = %+v, expected %+v", result, expected)
		}
	})

	t.Run("nested node", func(t *testing.T) {
		result := transformNode(nestedNode, nil)
		if result.Key != "parent" || result.Relatedness != 0.75 {
			t.Errorf("transformNode() key/relatedness = %s/%f, expected parent/0.75", result.Key, result.Relatedness)
		}

		// Check that we have one traversal
		if len(result.Traversals) != 1 {
			t.Fatalf("Expected 1 traversal, got %d", len(result.Traversals))
		}

		// Check the traversal's values
		traversal := result.Traversals[0]
		if len(traversal.Values) != 1 {
			t.Fatalf("Expected 1 value in traversal, got %d", len(traversal.Values))
		}

		// Check the child node
		childNode := traversal.Values[0]
		if childNode.Key != "child1" || childNode.Relatedness != 0.5 {
			t.Errorf("Child node = %s/%f, expected child1/0.5", childNode.Key, childNode.Relatedness)
		}
	})
}

func TestProcessBuckets(t *testing.T) {
	buckets := []interface{}{
		map[string]interface{}{
			"val":   "bucket1",
			"count": float64(100),
			"relatedness": map[string]interface{}{
				"relatedness": float64(0.75),
			},
		},
		map[string]interface{}{
			"val":   "bucket2",
			"count": float64(50),
			"relatedness": map[string]interface{}{
				"relatedness": float64(0.5),
			},
		},
		"not a map", // This should be skipped
	}

	result := processBuckets(buckets, nil)

	if len(result) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(result))
	}

	if result[0].Key != "bucket1" || result[0].Relatedness != 0.75 {
		t.Errorf("First node = %s/%f, expected bucket1/0.75", result[0].Key, result[0].Relatedness)
	}

	if result[1].Key != "bucket2" || result[1].Relatedness != 0.5 {
		t.Errorf("Second node = %s/%f, expected bucket2/0.5", result[1].Key, result[1].Relatedness)
	}

	// Test with invalid input
	invalidResult := processBuckets("not a slice", nil)
	if invalidResult != nil {
		t.Errorf("Expected nil for invalid input, got %v", invalidResult)
	}
}

func TestTransformResponseFacet(t *testing.T) {
	// Test with the provided JSON
	jsonStr := `{
		"facets":{
			"count":200000,
			"f0_0":{
				"count":9905,
				"relatedness":{
					"relatedness":0.0,
					"foreground_popularity":0.04953,
					"background_popularity":0.04953
				},
				"f1_0":{
					"buckets":[{
						"val":"画",
						"count":9894,
						"relatedness":{
							"relatedness":0.91892,
							"foreground_popularity":0.04947,
							"background_popularity":0.04947
						}
					},{
						"val":"像",
						"relatedness":{
							"relatedness":0.79354,
							"foreground_popularity":0.01326,
							"background_popularity":0.02334
						}
					},{
						"val":"面",
						"count":4768,
						"relatedness":{
							"relatedness":0.77828,
							"foreground_popularity":0.02384,
							"background_popularity":0.0782
						}
					},{
						"val":"映",
						"count":1833,
						"relatedness":{
							"relatedness":0.76403,
							"foreground_popularity":0.00917,
							"background_popularity":0.01485
						}
					},{
						"val":"漫",
						"count":349,
						"relatedness":{
							"relatedness":0.57913,
							"foreground_popularity":0.00175,
							"background_popularity":0.00175
						}
					},{
						"val":"録",
						"count":671,
						"relatedness":{
							"relatedness":0.50234,
							"foreground_popularity":0.00336,
							"background_popularity":0.00822
						}
					},{
						"val":"動",
						"count":2304,
						"relatedness":{
							"relatedness":0.48716,
							"foreground_popularity":0.01152,
							"background_popularity":0.07096
						}
					},{
						"val":"見",
						"count":3163,
						"relatedness":{
							"relatedness":0.47988,
							"foreground_popularity":0.01582,
							"background_popularity":0.11965
						}
					}]
				}
			}
		}
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	facets, ok := data["facets"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected facets to be a map, got %T", data["facets"])
	}

	result := transformResponseFacet(facets, nil)

	// Verify the result structure
	if len(result) != 1 {
		t.Fatalf("Expected 1 traversal, got %d", len(result))
	}

	f0Traversal, exists := result["f0"]
	if !exists {
		t.Fatalf("Expected traversal with key 'f0', not found in %v", result)
	}

	// Check f0 traversal values
	if len(f0Traversal.Values) != 1 {
		t.Fatalf("Expected 1 value in f0 traversal, got %d", len(f0Traversal.Values))
	}

	f0Node := f0Traversal.Values[0]
	if f0Node.Relatedness != 0.0 {
		t.Errorf("f0 node relatedness = %f, expected 0.0", f0Node.Relatedness)
	}

	// Check f1 traversal within f0 node
	if len(f0Node.Traversals) != 1 {
		t.Fatalf("Expected 1 traversal in f0 node, got %d", len(f0Node.Traversals))
	}

	f1Traversal := f0Node.Traversals[0]
	if len(f1Traversal.Values) != 8 {
		t.Fatalf("Expected 8 values in f1 traversal, got %d", len(f1Traversal.Values))
	}

	// Check a few of the f1 values
	expectedValues := []struct {
		key         string
		relatedness float64
	}{
		{"画", 0.91892},
		{"像", 0.79354},
		{"面", 0.77828},
	}

	for i, expected := range expectedValues {
		if i >= len(f1Traversal.Values) {
			t.Fatalf("Not enough values in f1 traversal")
		}

		actual := f1Traversal.Values[i]
		if actual.Key != expected.key || !almostEqual(actual.Relatedness, expected.relatedness, 0.00001) {
			t.Errorf("f1 value[%d] = %s/%f, expected %s/%f",
				i, actual.Key, actual.Relatedness, expected.key, expected.relatedness)
		}
	}
}

// TestIntegration tests the full transformation process with the provided JSON
func TestIntegration(t *testing.T) {
	jsonStr := `{
		"facets":{
			"count":200000,
			"f0_0":{
				"count":9905,
				"relatedness":{
					"relatedness":0.0,
					"foreground_popularity":0.04953,
					"background_popularity":0.04953
				},
				"f1_0":{
					"buckets":[{
						"val":"画",
						"count":9894,
						"relatedness":{
							"relatedness":0.91892,
							"foreground_popularity":0.04947,
							"background_popularity":0.04947
						}
					},{
						"val":"像",
						"relatedness":{
							"relatedness":0.79354,
							"foreground_popularity":0.01326,
							"background_popularity":0.02334
						}
					},{
						"val":"面",
						"count":4768,
						"relatedness":{
							"relatedness":0.77828,
							"foreground_popularity":0.02384,
							"background_popularity":0.0782
						}
					},{
						"val":"映",
						"count":1833,
						"relatedness":{
							"relatedness":0.76403,
							"foreground_popularity":0.00917,
							"background_popularity":0.01485
						}
					},{
						"val":"漫",
						"count":349,
						"relatedness":{
							"relatedness":0.57913,
							"foreground_popularity":0.00175,
							"background_popularity":0.00175
						}
					},{
						"val":"録",
						"count":671,
						"relatedness":{
							"relatedness":0.50234,
							"foreground_popularity":0.00336,
							"background_popularity":0.00822
						}
					},{
						"val":"動",
						"count":2304,
						"relatedness":{
							"relatedness":0.48716,
							"foreground_popularity":0.01152,
							"background_popularity":0.07096
						}
					},{
						"val":"見",
						"count":3163,
						"relatedness":{
							"relatedness":0.47988,
							"foreground_popularity":0.01582,
							"background_popularity":0.11965
						}
					}]
				}
			}
		}
	}`

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Expected structure after transformation
	expectedTraversals := map[string]skg.Traversal{
		"f0": {
			Values: []skg.Node{
				{
					Relatedness: 0.0,
					Traversals: []skg.Traversal{
						{
							Values: []skg.Node{
								{Key: "画", Relatedness: 0.91892},
								{Key: "像", Relatedness: 0.79354},
								{Key: "面", Relatedness: 0.77828},
								{Key: "映", Relatedness: 0.76403},
								{Key: "漫", Relatedness: 0.57913},
								{Key: "録", Relatedness: 0.50234},
								{Key: "動", Relatedness: 0.48716},
								{Key: "見", Relatedness: 0.47988},
							},
						},
					},
				},
			},
		},
	}

	// Transform the data
	facets, ok := data["facets"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected facets to be a map, got %T", data["facets"])
	}

	result := transformResponseFacet(facets, nil)

	// Compare the result with the expected structure
	// Note: We're only checking the structure and key values, not doing a deep equality check
	if len(result) != len(expectedTraversals) {
		t.Fatalf("Expected %d traversals, got %d", len(expectedTraversals), len(result))
	}

	for key, expectedTraversal := range expectedTraversals {
		actualTraversal, exists := result[key]
		if !exists {
			t.Fatalf("Expected traversal with key '%s', not found", key)
		}

		// Check values length
		if len(actualTraversal.Values) != len(expectedTraversal.Values) {
			t.Fatalf("Expected %d values in traversal '%s', got %d",
				len(expectedTraversal.Values), key, len(actualTraversal.Values))
		}

		// Check first value's traversals
		actualNode := actualTraversal.Values[0]
		expectedNode := expectedTraversal.Values[0]

		if !almostEqual(actualNode.Relatedness, expectedNode.Relatedness, 0.00001) {
			t.Errorf("Node relatedness = %f, expected %f", actualNode.Relatedness, expectedNode.Relatedness)
		}

		if len(actualNode.Traversals) != len(expectedNode.Traversals) {
			t.Fatalf("Expected %d traversals in node, got %d",
				len(expectedNode.Traversals), len(actualNode.Traversals))
		}

		// Check nested traversal values
		actualNestedTraversal := actualNode.Traversals[0]
		expectedNestedTraversal := expectedNode.Traversals[0]

		if len(actualNestedTraversal.Values) != len(expectedNestedTraversal.Values) {
			t.Fatalf("Expected %d values in nested traversal, got %d",
				len(expectedNestedTraversal.Values), len(actualNestedTraversal.Values))
		}

		// Check a few of the nested values
		for i, expectedValue := range expectedNestedTraversal.Values {
			actualValue := actualNestedTraversal.Values[i]
			if actualValue.Key != expectedValue.Key ||
				!almostEqual(actualValue.Relatedness, expectedValue.Relatedness, 0.00001) {
				t.Errorf("Nested value[%d] = %s/%f, expected %s/%f",
					i, actualValue.Key, actualValue.Relatedness,
					expectedValue.Key, expectedValue.Relatedness)
			}
		}
	}
}

// Helper function to compare float values with a tolerance
func almostEqual(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
