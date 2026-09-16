//go:build integration

package integrationenv

import (
	"context"
	"database/sql"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	mysqlclient "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
)

func localAddress(t *testing.T, addr string) {
	t.Helper()
	host, _, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	require.True(t, host == "127.0.0.1" || host == "localhost" || host == "::1", "integration services must use a loopback address")
}

// MySQL creates one database per test. Only that database is dropped on cleanup.
func MySQL(t *testing.T) *gorm.DB {
	t.Helper()
	raw := os.Getenv("FEEDBACK_INTEGRATION_MYSQL_DSN")
	require.NotEmpty(t, raw, "integration requires FEEDBACK_INTEGRATION_MYSQL_DSN")
	cfg, err := mysqlclient.ParseDSN(raw)
	require.NoError(t, err)
	require.Equal(t, "feedback98_test", cfg.DBName, "use the dedicated test DSN marker")
	require.Equal(t, "tcp", cfg.Net)
	localAddress(t, cfg.Addr)
	cfg.DBName = ""
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	cfg.ParseTime = true
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	name := "feedback98_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	t.Cleanup(func() {
		_, err := admin.Exec("DROP DATABASE IF EXISTS `" + name + "`")
		require.NoError(t, err)
		require.NoError(t, admin.Close())
	})
	_, err = admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4")
	require.NoError(t, err)
	cfg.DBName = name
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: gormlog.Default.LogMode(gormlog.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(12)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return db
}

// Redis never flushes a database; each test owns a random key namespace.
func Redis(t *testing.T) (*redis.Client, string) {
	t.Helper()
	addr := os.Getenv("FEEDBACK_INTEGRATION_REDIS_ADDR")
	require.NotEmpty(t, addr, "integration requires FEEDBACK_INTEGRATION_REDIS_ADDR")
	localAddress(t, addr)
	client := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1, DialTimeout: time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second})
	require.NoError(t, client.Ping(context.Background()).Err())
	prefix := "test:feedback98:" + uuid.NewString()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var cursor uint64
		for {
			keys, next, err := client.Scan(ctx, cursor, prefix+"*", 100).Result()
			require.NoError(t, err)
			if len(keys) > 0 {
				require.NoError(t, client.Del(ctx, keys...).Err())
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
		require.NoError(t, client.Close())
	})
	return client, prefix
}
