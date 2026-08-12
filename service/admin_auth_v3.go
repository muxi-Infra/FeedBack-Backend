package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"golang.org/x/crypto/bcrypt"
)

type AdminAuthServiceV3 interface {
	Login(ctx context.Context, input domain.AdminLoginInputV3) (domain.AdminLoginResultV3, error)
}

type adminAuthServiceV3 struct {
	users dao.AdminUserDAOV3
	jwt   *ijwt.AdminJWTV3
}

func NewAdminAuthServiceV3(users dao.AdminUserDAOV3, jwt *ijwt.AdminJWTV3) AdminAuthServiceV3 {
	return &adminAuthServiceV3{
		users: users,
		jwt:   jwt,
	}
}

func (s *adminAuthServiceV3) Login(ctx context.Context, input domain.AdminLoginInputV3) (domain.AdminLoginResultV3, error) {
	user, err := s.users.GetByUsername(ctx, strings.TrimSpace(input.Username))
	if err != nil {
		return domain.AdminLoginResultV3{}, errs.V3AdminDatabaseError(err)
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return domain.AdminLoginResultV3{}, errs.V3AdminCredentialsError(errors.New("invalid admin credentials"))
	}

	now := time.Now()
	if err := s.users.UpdateLoginInfo(ctx, user.ID, now); err != nil {
		return domain.AdminLoginResultV3{}, errs.V3AdminDatabaseError(err)
	}
	token, err := s.jwt.Issue(user.ID)
	if err != nil {
		return domain.AdminLoginResultV3{}, errs.V3TokenGenerateError(err)
	}
	return domain.AdminLoginResultV3{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   s.jwtTimeoutSeconds(),
	}, nil
}

func (s *adminAuthServiceV3) jwtTimeoutSeconds() int64 {
	return s.jwt.TTLSeconds()
}

// HashAdminPasswordV3 供初始化首个管理员的测试/脚本使用。
func HashAdminPasswordV3(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errs.V3PasswordHashError(err)
	}
	return string(hash), nil
}
