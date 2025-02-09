package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// SolrSetupParams はSolrのセットアップに必要なパラメータを定義します
type SolrSetupParams struct {
	CollectionName    string `json:"collectionName" validate:"required"`
	NumShards         int    `json:"numShards" validate:"required"`
	ReplicationFactor int    `json:"replicationFactor" validate:"required"`
}

// NewSetupSolrHandlerはApache Solrのセットアップを行うエンドポイントを返す
// SolrCloudのCollectionを作成し、必要な設定を行う
func NewSetupSolrHandler() func(echo.Context) error {
	return func(c echo.Context) error {

		var params SolrSetupParams
		if err := c.Bind(&params); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}

		solrURL := fmt.Sprintf("http://solr:8983/solr/admin/collections?action=CREATE&name=%s&numShards=%d&replicationFactor=%d",
			params.CollectionName, params.NumShards, params.ReplicationFactor)

		resp, err := http.Get(solrURL)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create collection"})
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Solr responded with an error"})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Collection created successfully"})
	}
}
