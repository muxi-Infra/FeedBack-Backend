package service

import (
	"context"
	"fmt"
	"sync"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/feishu"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/logger"
)

type TableService interface {
	RefreshTableConfig() ([]Table, error)
	GetTableConfig(tableIdentity string) (Table, error)
}

type TableServiceImpl struct {
	baseTableCfg *config.BaseTable
	tableCfg     map[string]Table
	mutex        sync.RWMutex
	c            feishu.Client
	log          logger.Logger
}

type Table struct {
	Identity   string
	Name       string
	TableToken string
	TableID    string
	ViewID     string
}

func NewTableService(cfg *config.BaseTable, c feishu.Client, log logger.Logger) TableService {
	return &TableServiceImpl{
		tableCfg:     make(map[string]Table),
		baseTableCfg: cfg,
		mutex:        sync.RWMutex{},
		c:            c,
		log:          log,
	}
}

func (t *TableServiceImpl) RefreshTableConfig() ([]Table, error) {
	// 创建请求对象
	req := larkbitable.NewSearchAppTableRecordReqBuilder().
		AppToken(t.baseTableCfg.TableToken).
		TableId(t.baseTableCfg.TableID).
		PageToken("").
		PageSize(50). // 分页大小，先给 50， 应该用不到这么多
		Body(larkbitable.NewSearchAppTableRecordReqBodyBuilder().
			ViewId(t.baseTableCfg.ViewID).
			FieldNames([]string{`table_identity`, `table_name`, `table_token`, `table_id`, `view_id`}).
			Build()).
		Build()

	// 发起请求
	ctx := context.Background()
	resp, err := t.c.GetAppTableRecord(ctx, req)

	// 处理错误
	if err != nil {
		t.log.Error("RefreshTableConfig 调用失败",
			logger.String("error", err.Error()),
		)
		return nil, errs.FeishuRequestError(err)
	}

	// 服务端错误处理
	if !resp.Success() {
		t.log.Error("RefreshTableConfig Lark 接口错误",
			logger.String("request_id", resp.RequestId()),
			logger.String("error", larkcore.Prettify(resp.CodeError)),
		)
		return nil, errs.FeishuResponseError(err)
	}

	var tables []Table
	for _, item := range resp.Data.Items {
		var table Table
		if item.Fields != nil {
			extract := func(key string) string {
				val, ok := item.Fields[key]
				if !ok || val == nil {
					return ""
				}

				switch v := val.(type) {
				case string:
					return v
				case []interface{}:
					if len(v) == 0 {
						return ""
					}
					elem := v[0]
					if s, ok := elem.(map[string]interface{}); ok {
						if txt, ok := s["text"]; ok {
							if ss, ok := txt.(string); ok {
								return ss
							}
						}
						return ""
					}
					return ""
				case map[string]interface{}:
					// 尝试获取 text 字段
					if txt, ok := v["text"]; ok {
						if s, ok := txt.(string); ok {
							return s
						}
					}
					return ""
				default:
					return fmt.Sprintf("%v", v)
				}
			}

			table.Identity = extract("table_identity")
			table.Name = extract("table_name")
			table.TableToken = extract("table_token")
			table.TableID = extract("table_id")
			table.ViewID = extract("view_id")
		}

		if table.Identity != "" {
			tables = append(tables, table)
		}
	}

	// 同步更新配置（在临界区内替换 map，避免并发读写风险）
	newTables := make(map[string]Table)
	for _, table := range tables {
		if table.Identity != "" {
			newTables[table.Identity] = table
		}
	}

	t.mutex.Lock()
	t.tableCfg = newTables
	t.mutex.Unlock()

	return tables, nil
}

func (t *TableServiceImpl) GetTableConfig(tableIdentity string) (Table, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	table, exists := t.tableCfg[tableIdentity]
	if !exists {
		return Table{},
			errs.TableIdentifyNotFoundError(fmt.Errorf("table identity %s not found", tableIdentity))
	}
	return table, nil
}
