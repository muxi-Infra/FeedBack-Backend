package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

const (
	faqIndexName = "feedback_faq_record_vector"

	// ✅ 已适配 FAQRecord 新结构
	faqMapping = `{
  "mappings": {
    "properties": {
      "vector": {
        "type": "dense_vector",
        "dims": 512
      },
      "id": { "type": "long" },
      "table_identify": { "type": "keyword" },
      "record_id":      { "type": "keyword" },
      "record":         { "type": "object", "enabled": true },
      "resolved_count":   { "type": "long" },
      "unresolved_count": { "type": "long" },
      "created_at":     { "type": "date" },
      "updated_at":     { "type": "date" }
    }
  }
}`
)

// FAQRecordDoc ES 文档结构（必须扁平）
type FAQRecordDoc struct {
	ID              uint64         `json:"id"`
	TableIdentify   *string        `json:"table_identify"`
	RecordID        *string        `json:"record_id"`
	Record          map[string]any `json:"record"`
	ResolvedCount   int64          `json:"resolved_count"`
	UnresolvedCount int64          `json:"unresolved_count"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`

	Vector []float64 `json:"vector"`
}

type FAQESRepo struct {
	client    *elasticsearch.Client
	indexName string
}

func NewFAQESRepo(client *elasticsearch.Client) (FAQESRepo, error) {
	repo := FAQESRepo{
		client:    client,
		indexName: faqIndexName,
	}

	if err := repo.ensureIndex(context.Background()); err != nil {
		return FAQESRepo{}, err
	}

	return repo, nil
}

func (r *FAQESRepo) ensureIndex(ctx context.Context) error {
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
		Body:  strings.NewReader(faqMapping),
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
func (r *FAQESRepo) SaveWithVector(ctx context.Context, data *model.FAQRecord, vector []float64) error {
	doc := &FAQRecordDoc{
		ID:              data.ID,
		TableIdentify:   data.TableIdentify,
		RecordID:        data.RecordID,
		Record:          data.Record,
		ResolvedCount:   data.ResolvedCount,
		UnresolvedCount: data.UnresolvedCount,
		CreatedAt:       data.CreatedAt,
		UpdatedAt:       data.UpdatedAt,
		Vector:          vector,
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

// SearchSimilarFAQ ES7 向量搜索（script_score）
func (r *FAQESRepo) SearchSimilarFAQ(ctx context.Context, vector []float64, topK int) ([]*model.FAQRecord, error) {
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
func (r *FAQESRepo) parseHits(body io.Reader) ([]*model.FAQRecord, error) {
	var response struct {
		Hits struct {
			Hits []struct {
				Source model.FAQRecord `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, err
	}

	results := make([]*model.FAQRecord, 0, len(response.Hits.Hits))

	for i := range response.Hits.Hits {
		results = append(results, &response.Hits.Hits[i].Source)
	}

	return results, nil
}
