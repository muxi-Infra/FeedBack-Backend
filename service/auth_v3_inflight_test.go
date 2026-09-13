package service_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/errorx"
	"github.com/muxi-Infra/FeedBack-Backend/repository/cache"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"github.com/muxi-Infra/FeedBack-Backend/service"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 仅控制数据库查询的完成时机，密钥查询仍由生产 DAO 执行。
type delayedExchangeKeyDAO struct {
	dao.IntegrationDAOV3
	wait func(context.Context) error
}

func (d delayedExchangeKeyDAO) GetKey(ctx context.Context, projectID, keyID string, tx ...*gorm.DB) (model.FeedbackProjectKeyV3, error) {
	key, err := d.IntegrationDAOV3.GetKey(ctx, projectID, keyID, tx...)
	if err == nil {
		err = d.wait(ctx)
	}
	return key, err
}

// 在实际 Redis 原子写入前后设置测试同步点，不替代 nonce 判重逻辑。
type delayedExchangeNonceStore struct {
	cache.IntegrationNonceStoreV3
	before func(context.Context) error
	after  func()
}

func (s delayedExchangeNonceStore) MarkUsed(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	if s.before != nil {
		if err := s.before(ctx); err != nil {
			return false, err
		}
	}
	used, err := s.IntegrationNonceStoreV3.MarkUsed(ctx, key, expiration)
	if s.after != nil {
		s.after()
	}
	return used, err
}

func newExchangeNonceClient(t *testing.T, f *securityFixture) cache.IntegrationNonceStoreV3 {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: f.redis.Addr(), MaxRetries: -1})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return cache.NewIntegrationNonceStoreV3(client)
}

func TestV3InFlightReplayAfterNonceExpiration(t *testing.T) {
	for _, phase := range []string{"database", "before_nonce_write"} {
		for _, offset := range []time.Duration{-300 * time.Second, 0, 300 * time.Second} {
			t.Run(phase+"/"+offset.String(), func(t *testing.T) {
				f := newSecurityFixture(t)
				start := time.Unix(1800000000, 0)
				var clockSeconds atomic.Int64
				clockSeconds.Store(start.Unix())
				now := func() time.Time { return time.Unix(clockSeconds.Load(), 0) }
				entered, release := make(chan struct{}), make(chan struct{})
				ctx, cancel := context.WithCancel(context.Background())
				wait := func(ctx context.Context) error {
					close(entered)
					select {
					case <-release:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				d := dao.NewIntegrationDAOV3(f.db)
				nonces := newExchangeNonceClient(t, f)
				if phase == "database" {
					d = delayedExchangeKeyDAO{IntegrationDAOV3: d, wait: wait}
				} else {
					nonces = delayedExchangeNonceStore{IntegrationNonceStoreV3: nonces, before: wait}
				}
				slowAuth := service.NewV3AuthService(d, nonces, inertProjectEvents{}, service.NewProjectConfigCacheV3(), f.jwt,
					&config.IntegrationAuthConfig{TimestampSkew: 300})
				input := exchangeInput(signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "inflight-nonce", start.Add(offset).Unix()))
				var token string
				var expires int64
				var exchangeErr error
				done := make(chan struct{})
				go func() {
					defer close(done)
					token, expires, exchangeErr = service.ExchangeV3WithClockForTest(slowAuth, ctx, input, now)
				}()
				// 即使断言失败，也要释放等待中的请求，再关闭数据库和 Redis。
				t.Cleanup(func() {
					cancel()
					select {
					case <-done:
					case <-time.After(5 * time.Second):
						t.Error("等待中的兑换请求未能退出")
					}
				})
				select {
				case <-entered:
				case <-time.After(5 * time.Second):
					t.Fatal("兑换请求未到达指定的等待位置")
				}

				// 两个请求均在有效期内开始；快速请求先兑换成功，慢请求仍在等待。
				firstToken, _, err := service.ExchangeV3WithClockForTest(f.auth, ctx, input, now)
				require.NoError(t, err)
				require.NotEmpty(t, firstToken)
				_, err = f.jwt.Parse(firstToken)
				require.NoError(t, err)
				elapsed := 300*time.Second + offset + time.Second
				f.redis.FastForward(elapsed)
				clockSeconds.Add(int64(elapsed / time.Second))
				require.Empty(t, f.redis.Keys(), "快速请求写入的 nonce 应已过期")
				close(release)
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Fatal("释放等待后兑换请求未完成")
				}
				require.Error(t, exchangeErr, "相同签名请求不能跨 nonce 过期再次兑换")
				require.Equal(t, errs.V3ExchangeExpiredCode, errorx.ToCustomError(exchangeErr).Code)
				require.Empty(t, token)
				require.Zero(t, expires)

				// 时间已推进后，全新且有效的兑换请求仍应成功。
				fresh := exchangeInput(signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "fresh-after-delay", now().Unix()))
				freshToken, _, err := service.ExchangeV3WithClockForTest(f.auth, ctx, fresh, now)
				require.NoError(t, err)
				require.NotEmpty(t, freshToken)
			})
		}
	}
}

func TestV3ExchangeRechecksTimeAfterNonceReply(t *testing.T) {
	for _, tc := range []struct {
		name    string
		delay   time.Duration
		expired bool
	}{
		{"last_valid_second", time.Second, false},
		{"expired_during_reply", 2 * time.Second, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSecurityFixture(t)
			now := time.Unix(1800000000, 0)
			input := exchangeInput(signedExchange(projectA, studentA, projectA+"-key", testAPIKey, "delayed-reply-nonce", now.Add(-299*time.Second).Unix()))
			nonces := delayedExchangeNonceStore{
				IntegrationNonceStoreV3: newExchangeNonceClient(t, f),
				after: func() {
					// 模拟 Redis 已完成写入，但应用稍后才收到结果。
					f.redis.FastForward(tc.delay)
					now = now.Add(tc.delay)
				},
			}
			auth := service.NewV3AuthService(dao.NewIntegrationDAOV3(f.db), nonces, inertProjectEvents{},
				service.NewProjectConfigCacheV3(), f.jwt, &config.IntegrationAuthConfig{TimestampSkew: 300})
			token, expires, err := service.ExchangeV3WithClockForTest(auth, context.Background(), input, func() time.Time { return now })
			if tc.expired {
				require.Error(t, err, "Redis 响应返回时已过期的请求不能签发令牌")
				require.Equal(t, errs.V3ExchangeExpiredCode, errorx.ToCustomError(err).Code)
				require.Empty(t, token)
				require.Zero(t, expires)
			} else {
				require.NoError(t, err)
				require.Positive(t, expires)
				claims, err := f.jwt.Parse(token)
				require.NoError(t, err)
				require.Equal(t, projectA, claims.ProjectID)
				require.Equal(t, studentA, claims.StudentID)
			}
		})
	}
}
