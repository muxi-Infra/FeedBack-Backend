package session

import (
	"context"

	"github.com/gin-gonic/gin"
)

type Session string

const TableIdentity Session = "tableIdentity"

func Set(c *gin.Context, key string, value interface{}) {
	c.Set(key, value)
}

func SetTableIdentity(c *gin.Context, value string) {
	c.Set(TableIdentity, value)
}

func GetTableIdentity(c context.Context) string {
	s, ok := c.Value(TableIdentity).(string)
	if !ok {
		return ""
	}

	return s
}
