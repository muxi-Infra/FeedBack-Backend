package middleware

import (
	"github.com/muxi-Infra/FeedBack-Backend/config"

	"github.com/gin-gonic/gin"
)

type BasicAuthMiddleware struct {
	accounts gin.Accounts
}

func NewBasicAuthMiddleware(basicUsers []config.BasicAuthConfig) *BasicAuthMiddleware {
	accounts := gin.Accounts{}
	for _, u := range basicUsers {
		accounts[u.Username] = u.Password
	}

	return &BasicAuthMiddleware{
		accounts: accounts,
	}
}

func (bm *BasicAuthMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return gin.BasicAuth(bm.accounts)
}
