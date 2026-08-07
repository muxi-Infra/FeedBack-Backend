package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/domain"
	"github.com/muxi-Infra/FeedBack-Backend/errs"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	"github.com/muxi-Infra/FeedBack-Backend/repository/dao"
	"github.com/muxi-Infra/FeedBack-Backend/repository/model"
	"golang.org/x/crypto/bcrypt"
)

const (
	adminMinPasswordLength = 8
)

//go:generate mockgen -destination=./mock/admin_mock.go -package=mocks github.com/muxi-Infra/FeedBack-Backend/service AdminService
type AdminService interface {
	CreateAdmin(ctx context.Context, input domain.CreateAdminInput) (domain.AdminUser, error)
	Login(ctx context.Context, input domain.AdminLoginInput) (domain.AdminLoginResult, error)
	GetAdmin(ctx context.Context, id uint64) (domain.AdminUser, error)
	ChangePassword(ctx context.Context, input domain.ChangeAdminPasswordInput) error
	UpdateStatus(ctx context.Context, id uint64, status string) error
	ParseToken(token string) (domain.AdminTokenClaims, error)
}

type adminService struct {
	users dao.AdminUserDAO
	jwt   *ijwt.AdminJWT
}

func NewAdminService(users dao.AdminUserDAO, adminJWT *ijwt.AdminJWT) AdminService {
	return &adminService{users: users, jwt: adminJWT}
}

func (s *adminService) CreateAdmin(ctx context.Context, input domain.CreateAdminInput) (domain.AdminUser, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if err := validateAdminCredentials(input.Username, input.Password); err != nil {
		return domain.AdminUser{}, errs.AdminInvalidInputError(err)
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Username
	}

	existing, err := s.users.GetByUsername(ctx, input.Username)
	if err != nil {
		return domain.AdminUser{}, errs.AdminDatabaseError(err)
	}
	if existing != nil {
		return domain.AdminUser{}, errs.AdminAlreadyExistsError(errors.New("admin username already exists"))
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AdminUser{}, errs.AdminPasswordError(err)
	}

	user := &model.AdminUser{
		Username:     input.Username,
		DisplayName:  input.DisplayName,
		PasswordHash: string(hash),
		Status:       "active",
	}
	if err := s.users.Create(ctx, user); err != nil {
		return domain.AdminUser{}, errs.AdminDatabaseError(err)
	}
	return toAdminUser(user), nil
}

func (s *adminService) Login(ctx context.Context, input domain.AdminLoginInput) (domain.AdminLoginResult, error) {
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" || input.Password == "" {
		return domain.AdminLoginResult{}, errs.AdminAuthFailedError(errors.New("username or password is invalid"))
	}

	user, err := s.users.GetByUsername(ctx, input.Username)
	if err != nil {
		return domain.AdminLoginResult{}, errs.AdminDatabaseError(err)
	}
	// 不区分“用户不存在”和“密码错误”，避免泄露账号是否存在。
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return domain.AdminLoginResult{}, errs.AdminAuthFailedError(errors.New("username or password is invalid"))
	}
	if user.Status != "active" {
		return domain.AdminLoginResult{}, errs.AdminDisabledError(errors.New("admin account is not active"))
	}

	now := time.Now()
	if err := s.users.UpdateLoginInfo(ctx, user.ID, now); err != nil {
		return domain.AdminLoginResult{}, errs.AdminDatabaseError(err)
	}
	user.LastLoginAt = &now

	token, expiresAt, err := s.issueToken(user)
	if err != nil {
		return domain.AdminLoginResult{}, errs.AdminTokenError(err)
	}
	return domain.AdminLoginResult{
		Token: token, ExpiresAt: expiresAt, User: toAdminUser(user),
	}, nil
}

func (s *adminService) GetAdmin(ctx context.Context, id uint64) (domain.AdminUser, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return domain.AdminUser{}, errs.AdminDatabaseError(err)
	}
	if user == nil {
		return domain.AdminUser{}, errs.AdminNotFoundError(errors.New("admin user not found"))
	}
	return toAdminUser(user), nil
}

func (s *adminService) ChangePassword(ctx context.Context, input domain.ChangeAdminPasswordInput) error {
	if input.AdminID == 0 {
		return errs.AdminInvalidInputError(errors.New("admin id is required"))
	}
	if err := validatePassword(input.NewPassword); err != nil {
		return errs.AdminInvalidInputError(err)
	}

	user, err := s.users.GetByID(ctx, input.AdminID)
	if err != nil {
		return errs.AdminDatabaseError(err)
	}
	if user == nil {
		return errs.AdminNotFoundError(errors.New("admin user not found"))
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.CurrentPassword)) != nil {
		return errs.AdminAuthFailedError(errors.New("current password is invalid"))
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errs.AdminPasswordError(err)
	}
	if err := s.users.UpdatePassword(ctx, input.AdminID, string(hash)); err != nil {
		return errs.AdminDatabaseError(err)
	}
	return nil
}

func (s *adminService) UpdateStatus(ctx context.Context, id uint64, status string) error {
	status = strings.TrimSpace(status)
	if id == 0 || status == "" {
		return errs.AdminInvalidInputError(errors.New("admin id and status are required"))
	}
	if status != "active" && status != "locked" && status != "disabled" {
		return errs.AdminInvalidInputError(fmt.Errorf("unsupported admin status: %s", status))
	}
	if err := s.users.UpdateStatus(ctx, id, status); err != nil {
		return errs.AdminDatabaseError(err)
	}
	return nil
}

func (s *adminService) issueToken(user *model.AdminUser) (string, time.Time, error) {
	return s.jwt.Issue(user.ID)
}

func (s *adminService) ParseToken(tokenString string) (domain.AdminTokenClaims, error) {
	claims, err := s.jwt.Parse(tokenString)
	if err != nil {
		return domain.AdminTokenClaims{}, errs.AdminTokenError(err)
	}
	if claims.Subject == "" || claims.ExpiresAt == nil {
		return domain.AdminTokenClaims{}, errs.AdminTokenError(errors.New("admin token claims are incomplete"))
	}
	adminID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || adminID == 0 {
		return domain.AdminTokenClaims{}, errs.AdminTokenError(errors.New("admin token subject is not a valid admin id"))
	}
	return domain.AdminTokenClaims{
		AdminID:   adminID,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func validateAdminCredentials(username, password string) error {
	if username == "" {
		return errors.New("username is required")
	}
	return validatePassword(password)
}

func validatePassword(password string) error {
	if len([]rune(password)) < adminMinPasswordLength {
		return fmt.Errorf("password must contain at least %d characters", adminMinPasswordLength)
	}
	return nil
}

func toAdminUser(user *model.AdminUser) domain.AdminUser {
	return domain.AdminUser{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Status:      user.Status,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
