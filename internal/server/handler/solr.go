package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
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

// NewSetupSolrSchemaHandler はSolrのコレクションのschemaを設定するエンドポイントを返します
func NewSetupSolrSchemaHandler() func(c echo.Context) error {
	return func(c echo.Context) error {
		var params SolrSchemaParams
		if err := c.Bind(&params); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}

		// Solr Schema API のエンドポイント
		solrSchemaURL := fmt.Sprintf("http://solr:8983/solr/%s/schema", params.CollectionName)

		// 追加フィールドの定義をpayloadとして作成
		payload := map[string]interface{}{
			"add-field": params.Fields,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to marshal schema payload"})
		}

		resp, err := http.Post(solrSchemaURL, "application/json", bytes.NewBuffer(payloadBytes))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update schema"})
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Solr responded with an error on schema update"})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Schema updated successfully"})
	}
}

// NewFeedSolrDataHandler はアップロードされたJSONファイルを読み込み、Solrのupdate APIにデータをフィードするエンドポイントを返します
func NewFeedSolrDataHandler() func(c echo.Context) error {
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
		solrUpdateURL := fmt.Sprintf("http://solr:8983/solr/%s/update?commit=true", collectionName)

		resp, err := http.Post(solrUpdateURL, "application/json", bytes.NewBuffer(fileBytes))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to feed data to Solr"})
		}
		defer resp.Body.Close()

		// 応答内容を取得（必要に応じて解析可能）
		var solrResp interface{}
		if err := json.NewDecoder(resp.Body).Decode(&solrResp); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode Solr response"})
		}

		if resp.StatusCode != http.StatusOK {
			return c.JSON(http.StatusInternalServerError, solrResp)
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Data fed to Solr successfully"})
	}
}
