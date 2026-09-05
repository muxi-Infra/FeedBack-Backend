package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/apikey"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
)

type V3ExchangeInput struct {
	ProjectID string
	KeyID     string
	StudentID string
	Timestamp int64
	Nonce     string
	Signature string
}

type V3TableConfig struct {
	ProjectID string
	TableType string
	Scopes    []string
	Table     model.FeedbackProjectTableV3
}

func (c V3TableConfig) Legacy() domain.TableConfig {
	identity, name, token, tableID, viewID := c.Table.TableIdentity, c.Table.PhysicalName, c.Table.TableToken, c.Table.TableID, c.Table.ViewID
	return domain.TableConfig{
		TableIdentity: &identity,
		TableName:     &name,
		TableToken:    &token,
		TableID:       &tableID,
		ViewID:        &viewID,
		Notice:        c.Table.Notice,
	}
}

func (c V3TableConfig) HasScope(scope string) bool {
	for _, item := range c.Scopes {
		if strings.TrimSpace(item) == scope {
			return true
		}
	}
	return false
}

type V3AuthService interface {
	Exchange(ctx context.Context, input V3ExchangeInput) (string, int64, error)
	GetTableConfig(ctx context.Context, projectID, tableType string) (V3TableConfig, error)
}

type v3AuthService struct {
	dao         dao.IntegrationDAOV3
	nonces      cache.IntegrationNonceStoreV3
	configCache *ProjectConfigCacheV3
	jwt         *ijwt.V3JWT
	config      *config.IntegrationAuthConfig
}

func NewV3AuthService(d dao.IntegrationDAOV3, n cache.IntegrationNonceStoreV3, events cache.ProjectConfigEventBusV3, localCache *ProjectConfigCacheV3, jwt *ijwt.V3JWT, cfg *config.IntegrationAuthConfig) V3AuthService {
	s := &v3AuthService{
		dao:         d,
		nonces:      n,
		configCache: localCache,
		jwt:         jwt,
		config:      cfg,
	}

	go events.ConsumeProjectChanged(context.Background(), func(projectID string) error {
		localCache.deleteProject(projectID)
		return nil
	})
	return s
}

func (s *v3AuthService) Exchange(ctx context.Context, input V3ExchangeInput) (string, int64, error) {
	return s.exchangeAt(ctx, input, time.Now())
}

func (s *v3AuthService) exchangeAt(ctx context.Context, input V3ExchangeInput, now time.Time) (string, int64, error) {
	if input.ProjectID == "" || input.KeyID == "" || input.StudentID == "" || input.Nonce == "" || input.Signature == "" {
		return "", 0, errs.V3InvalidInputError(errors.New("v3 exchange fields are required"))
	}

	window := 300
	if s.config != nil && s.config.TimestampSkew > 0 {
		window = s.config.TimestampSkew
	}
	delta := now.Unix() - input.Timestamp
	if delta > int64(window) || delta < -int64(window) {
		return "", 0, errs.V3ExchangeExpiredError(errors.New("v3 exchange timestamp is expired"))
	}

	project, err := s.dao.GetProject(ctx, input.ProjectID)
	if err != nil {
		return "", 0, errs.V3ProjectLookupError(err)
	}
	key, err := s.dao.GetKey(ctx, project.ProjectID, input.KeyID)
	if err != nil {
		return "", 0, errs.V3APIKeyInvalidError(err)
	}
	payload := strings.Join([]string{
		input.ProjectID,
		input.KeyID,
		input.StudentID,
		strconv.FormatInt(input.Timestamp, 10),
		input.Nonce,
	}, "\n")
	if err := apikey.Verify(key.APIKeyHash, payload, input.Signature); err != nil {
		return "", 0, errs.V3SignatureInvalidError(err)
	}

	// nonce 必须保留到请求的最后一个有效秒结束。
	// 携带未来时间戳的请求，首次使用后的剩余有效期可能超过一个时间偏差窗口。
	nonceTTL := time.Duration(int64(window)-delta+1) * time.Second
	used, err := s.nonces.MarkUsed(ctx, "v3:exchange:nonce:"+input.ProjectID+":"+input.Nonce, nonceTTL)
	if err != nil {
		return "", 0, errs.V3NonceError(err)
	}
	if !used {
		return "", 0, errs.V3ReplayRequestError(errors.New("v3 exchange nonce has already been used"))
	}

	token, expires, err := s.jwt.Issue(input.ProjectID, input.StudentID)
	if err != nil {
		return "", 0, errs.V3TokenGenerateError(err)
	}
	return token, expires, nil
}

func (s *v3AuthService) GetTableConfig(ctx context.Context, projectID, tableType string) (V3TableConfig, error) {
	if projectID == "" || tableType == "" {
		return V3TableConfig{}, errs.V3InvalidInputError(errors.New("project_id and table_type are required"))
	}

	if cached, ok := s.configCache.get(projectID, tableType); ok {
		return cached, nil
	}

	if _, err := s.dao.GetProject(ctx, projectID); err != nil {
		return V3TableConfig{}, errs.V3TableConfigError(err)
	}

	table, err := s.dao.GetTable(ctx, projectID, tableType)
	if err != nil {
		return V3TableConfig{}, errs.V3TableConfigError(err)
	}

	scopes, err := s.dao.ListScopes(ctx, projectID, table.TableIdentity)
	if err != nil {
		return V3TableConfig{}, errs.V3TableConfigError(err)
	}

	result := V3TableConfig{
		ProjectID: projectID,
		TableType: tableType,
		Scopes:    scopes,
		Table:     table,
	}
	s.configCache.set(result)

	return result, nil
}
