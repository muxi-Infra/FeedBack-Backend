package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/muxi-Infra/FeedBack-Backend/repository/model"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

// 定义索引名称和 Mapping 常量
const (
	faqIndexName = "feedback_faq_resolution_vector"
	faqMapping   = `{
  "mappings": {
    "properties": {
      "vector": {
        "type": "dense_vector",
        "dims": 1024,
        "index": true,
        "similarity": "cosine"
      },
      "user_id":        { "type": "keyword" },
      "table_identify": { "type": "keyword" },
      "record_id":      { "type": "keyword" },
      "is_resolved":    { "type": "boolean" },
      "frequency":      { "type": "integer" },
      "created_at":     { "type": "date" },
      "updated_at":     { "type": "date" }
    }
  }
}`
)

type FAQResolutionDoc struct {
	model.FAQResolution
	Vector []float64 `json:"vector,omitempty"`
}

type FAQESRepo struct {
	client    *elasticsearch.Client
	indexName string
}

func NewFAQESRepo(client *elasticsearch.Client) (*FAQESRepo, error) {
	repo := &FAQESRepo{
		client:    client,
		indexName: faqIndexName,
	}

	// 初始化检查
	if err := repo.ensureIndex(context.Background()); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *FAQESRepo) ensureIndex(ctx context.Context) error {
	// 检查是否存在
	res, err := r.client.Indices.Exists([]string{r.indexName})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil // 已经有了，溜了
	}

	// 不存在则直接用常量创建
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

// SaveWithVector 写入
func (r *FAQESRepo) SaveWithVector(ctx context.Context, data *model.FAQResolution, vector []float64) error {
	doc := &FAQResolutionDoc{
		FAQResolution: *data,
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

// SearchSimilarFAQ 搜索
func (r *FAQESRepo) SearchSimilarFAQ(ctx context.Context, vector []float64, topK int) ([]*model.FAQResolution, error) {
	query := map[string]any{
		"knn": map[string]any{
			"field":          "vector",
			"query_vector":   vector,
			"k":              topK,
			"num_candidates": 50,
		},
		"_source": []string{
			"id", "user_id", "table_identify", "record_id", "is_resolved", "frequency", "created_at", "updated_at",
		},
	}

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(query)

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

func (r *FAQESRepo) parseHits(body io.Reader) ([]*model.FAQResolution, error) {
	var response struct {
		Hits struct {
			Hits []struct {
				Source model.FAQResolution `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return nil, err
	}

	results := make([]*model.FAQResolution, 0, len(response.Hits.Hits))

	for i := range response.Hits.Hits {
		results = append(results, &response.Hits.Hits[i].Source)
	}

	return results, nil
}
