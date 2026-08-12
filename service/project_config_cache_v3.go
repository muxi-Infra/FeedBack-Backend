package service

import (
	"sync"
)

// projectConfigCacheKey 标识一个项目下的一类表格配置。
type projectConfigCacheKey struct {
	ProjectID string
	TableType string
}

type ProjectConfigCacheV3 struct {
	mu    sync.RWMutex
	items map[projectConfigCacheKey]V3TableConfig
}

func NewProjectConfigCacheV3() *ProjectConfigCacheV3 {
	return &ProjectConfigCacheV3{
		items: make(map[projectConfigCacheKey]V3TableConfig),
	}
}

func (c *ProjectConfigCacheV3) get(projectID, tableType string) (V3TableConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[projectConfigCacheKey{
		ProjectID: projectID,
		TableType: tableType,
	}]
	return item, ok
}

func (c *ProjectConfigCacheV3) set(item V3TableConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[projectConfigCacheKey{
		ProjectID: item.ProjectID,
		TableType: item.TableType,
	}] = item
}

func (c *ProjectConfigCacheV3) deleteProject(projectID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.items {
		if key.ProjectID == projectID {
			delete(c.items, key)
		}
	}
}
