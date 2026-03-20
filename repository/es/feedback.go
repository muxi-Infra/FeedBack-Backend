package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

const (
	feedbackIndexName = "feedback_feedback_record_vector"

	feedbackMapping = `{
  "mappings": {
    "properties": {
      "vector": {
        "type": "dense_vector",
        "dims": 512
      },
      "id": { "type": "long" },
      "table_identify": { "type": "keyword" },
      "record_id":      { "type": "keyword" },
      "user_id":        { "type": "long" },
      "record":         { "type": "object", "enabled": true },
      "resolved_count":   { "type": "long" },
      "unresolved_count": { "type": "long" },
      "created_at":     { "type": "date" },
      "updated_at":     { "type": "date" }
    }
  }
}`
)

// FeedbackRecordDoc ES 文档结构（必须扁平）
type FeedbackRecordDoc struct {
	ID            uint64         `json:"id"`
	TableIdentify *string        `json:"table_identify"`
	RecordID      *string        `json:"record_id"`
	UserID        *string        `json:"user_id"`
	Record        map[string]any `json:"record"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`

	Vector []float64 `json:"vector"`
}

type FeedbackESRepo struct {
	client    *elasticsearch.Client
	indexName string
}

func NewFeedbackESRepo(client *elasticsearch.Client) (FeedbackESRepo, error) {
	repo := FeedbackESRepo{
		client:    client,
		indexName: feedbackIndexName,
	}

	if err := repo.ensureIndex(context.Background()); err != nil {
		return FeedbackESRepo{}, err
	}

	return repo, nil
}

func (r *FeedbackESRepo) ensureIndex(ctx context.Context) error {
	// 方便测试的时候清除非法结构
	//_, err := r.client.Indices.Delete([]string{r.indexName})
	//if err != nil {
	//	return err
	//}

	res, err := r.client.Indices.Exists([]string{r.indexName})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil
	}

	req := esapi.IndicesCreateRequest{
		Index: r.indexName,
		Body:  strings.NewReader(feedbackMapping),
	}

	createRes, err := req.Do(ctx, r.client)
	if err != nil {
		return err
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("failed to create es index: %s", createRes.String())
	}

	return nil
}

// SaveWithVector 写入 ES（带向量）
func (r *FeedbackESRepo) SaveWithVector(ctx context.Context, data *model.Sheet, vector []float64) error {
	doc := &FeedbackRecordDoc{
		ID:            data.ID,
		TableIdentify: data.TableIdentify,
		RecordID:      data.RecordID,
		UserID:        data.UserID,
		Record:        data.Record,
		CreatedAt:     data.CreatedAt,
		UpdatedAt:     data.UpdatedAt,
		Vector:        vector,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index:      r.indexName,
		DocumentID: fmt.Sprintf("%d", data.ID),
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("index error: %s", res.String())
	}

	return nil
}

// SearchSimilarFeedback ES7 向量搜索（script_score）
func (r *FeedbackESRepo) SearchSimilarFeedback(ctx context.Context, vector []float64, topK int) ([]*model.Sheet, error) {
	query := map[string]any{
		"size": topK,
		"query": map[string]any{
			"script_score": map[string]any{
				"query": map[string]any{
					"match_all": map[string]any{},
				},
				"script": map[string]any{
					"source": "cosineSimilarity(params.query_vector, 'vector') + 1.0",
					"params": map[string]any{
						"query_vector": vector,
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexName),
		r.client.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search error: %s", res.String())
	}

	return r.parseHits(res.Body)
}

// 解析返回
func (r *FeedbackESRepo) parseHits(body io.Reader) ([]*model.Sheet, error) {
	var response struct {
		Hits struct {
			Hits []struct {
				Source model.Sheet `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, err
	}

	results := make([]*model.Sheet, 0, len(response.Hits.Hits))

	for i := range response.Hits.Hits {
		results = append(results, &response.Hits.Hits[i].Source)
	}

	return results, nil
}
