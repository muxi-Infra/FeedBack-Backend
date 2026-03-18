package ioc

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/llm"
)

func InitChatModel(cfg *config.LLMConfig) (model.ToolCallingChatModel, error) {
	ctx := context.Background()

	// 转换配置格式
	aiCfg := &llm.Config{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
	}

	m, err := llm.NewChatModel(ctx, aiCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize chat model: %w", err)
	}

	return m, nil
}

// InitLocalEmbedder 初始化本地向量化模型 (gte-small-zh)
func InitLocalEmbedder(cfg *config.LLMConfig) (embedding.Embedder, error) {
	ctx := context.Background()
	if cfg.EmbedURL == "" {
		return nil, fmt.Errorf("local model path is required for embedding")
	}

	emb, err := llm.NewLocalPureGoEmbedder(ctx, cfg.EmbedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize local embedder: %w", err)
	}

	return emb, nil
}
