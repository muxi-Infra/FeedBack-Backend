package ioc

import (
	"fmt"

	"github.com/casbin/casbin/v3"
	casbinmodel "github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

const adminCasbinModelV3 = `
[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act`

// InitCasbinV3 初始化 V3 管理员权限，策略保存在 MySQL 的 casbin_rule 表中。
func InitCasbinV3(db *gorm.DB) (*casbin.Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("初始化 V3 Casbin Adapter 失败: %w", err)
	}
	model, err := casbinmodel.NewModelFromString(adminCasbinModelV3)
	if err != nil {
		return nil, err
	}
	enforcer, err := casbin.NewEnforcer(model, adapter)
	if err != nil {
		return nil, err
	}
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, err
	}
	return enforcer, nil
}
