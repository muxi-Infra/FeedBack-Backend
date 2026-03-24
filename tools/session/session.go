package session

import "github.com/gin-gonic/gin"

type Session string

const TableIdentity Session = "tableIdentity"

func Set(c *gin.Context, key string, value interface{}) {
	c.Set(key, value)
}

func SetTableIdentity(c *gin.Context, value string) {
	c.Set(TableIdentity, value)
}

func GetTableIdentity(c *gin.Context) string {
	s, ok := c.Get(TableIdentity)
	if !ok {
		return ""
	}

	return s.(string)
}
