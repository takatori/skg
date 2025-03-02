package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/takatori/skg/internal"
	"github.com/takatori/skg/internal/infra"
)

// SolrSetupParams はSolrのセットアップに必要なパラメータを定義します
type SolrSetupParams struct {
	CollectionName    string `json:"collectionName" validate:"required"`
	NumShards         int    `json:"numShards" validate:"required"`
	ReplicationFactor int    `json:"replicationFactor" validate:"required"`
}

// SolrSchemaField はコレクションのschemaに追加するフィールド定義を表します
type SolrSchemaField struct {
	Name        string `json:"name" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Stored      bool   `json:"stored"`
	Indexed     bool   `json:"indexed"`
	MultiValued bool   `json:"multiValued"`
}

// SolrSchemaParams はスキーマ設定に必要なパラメータを定義します
type SolrSchemaParams struct {
	CollectionName string            `json:"collectionName" validate:"required"`
	Fields         []SolrSchemaField `json:"fields" validate:"required,dive,required"`
}

// SolrHandler はSolr関連のハンドラを提供する構造体です
type SolrHandler struct {
	config     *internal.Config
	httpClient *infra.HttpClient
}

// NewSolrHandler は新しいSolrHandlerを作成します
func NewSolrHandler(config *internal.Config, httpClient *infra.HttpClient) *SolrHandler {
	return &SolrHandler{
		config:     config,
		httpClient: httpClient,
	}
}

// SetupSolrHandler はApache Solrのセットアップを行うエンドポイントを返します
// SolrCloudのCollectionを作成し、必要な設定を行います
func (h *SolrHandler) SetupSolrHandler() func(echo.Context) error {
	return func(c echo.Context) error {
		var params SolrSetupParams
		if err := c.Bind(&params); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}

		solrURL := fmt.Sprintf("%s/admin/collections?action=CREATE&name=%s&numShards=%d&replicationFactor=%d",
			h.config.SolrUrl,
			params.CollectionName, params.NumShards, params.ReplicationFactor)

		// Use the HTTP client to make the GET request
		var solrResp map[string]interface{}
		err := h.httpClient.Get(
			c.Request().Context(),
			infra.Request{
				Url: solrURL,
			},
			&solrResp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create collection"})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Collection created successfully"})
	}
}

// SetupSolrSchemaHandler はSolrのコレクションのschemaを設定するエンドポイントを返します
func (h *SolrHandler) SetupSolrSchemaHandler() func(c echo.Context) error {
	return func(c echo.Context) error {
		var params SolrSchemaParams
		if err := c.Bind(&params); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}

		// Solr Schema API のエンドポイント
		solrSchemaURL := fmt.Sprintf("%s/%s/schema", h.config.SolrUrl, params.CollectionName)

		// 追加フィールドの定義をpayloadとして作成
		payload := map[string]interface{}{
			"add-field": params.Fields,
		}

		// Create a response map to hold the Solr response
		var solrResp map[string]interface{}

		// Use the HTTP client to make the request
		err := h.httpClient.Post(
			c.Request().Context(),
			infra.PostRequest{
				Request: infra.Request{
					Url: solrSchemaURL,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
				Entity: payload,
			},
			&solrResp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update schema"})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Schema updated successfully"})
	}
}

// FeedSolrDataHandler はアップロードされたJSONファイルを読み込み、Solrのupdate APIにデータをフィードするエンドポイントを返します
func (h *SolrHandler) FeedSolrDataHandler() func(c echo.Context) error {
	return func(c echo.Context) error {
		collectionName := c.FormValue("collectionName")
		if collectionName == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "collectionName is required"})
		}

		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
		}

		f, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to open file"})
		}
		defer f.Close()

		fileBytes, err := io.ReadAll(f)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read file"})
		}

		// Solr update API のエンドポイント (commit=true)
		solrUpdateURL := fmt.Sprintf("%s/%s/update?commit=true", h.config.SolrUrl, collectionName)

		// Parse the file bytes into a JSON object
		var jsonData interface{}
		if err := json.Unmarshal(fileBytes, &jsonData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse JSON file"})
		}

		// Create a response map to hold the Solr response
		var solrResp interface{}

		// Use the HTTP client to make the request
		err = h.httpClient.Post(
			c.Request().Context(),
			infra.PostRequest{
				Request: infra.Request{
					Url: solrUpdateURL,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
				Entity: jsonData,
			},
			&solrResp,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to feed data to Solr"})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Data fed to Solr successfully"})
	}
}
