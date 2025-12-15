package controller

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	FeishuMock "github.com/muxi-Infra/FeedBack-Backend/pkg/feishu/mock"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	LoggerMock "github.com/muxi-Infra/FeedBack-Backend/pkg/logger/mock"
	AuthMock "github.com/muxi-Infra/FeedBack-Backend/service/mock"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/stretchr/testify/assert"
)

// 创建一个mock的Sheet对象
func NewMockSheet(crtl *gomock.Controller) (*Sheet, *AuthMock.MockAuthService, *FeishuMock.MockClient, *LoggerMock.MockLogger) {
	mockAuth := AuthMock.NewMockAuthService(crtl)
	mockClient := FeishuMock.NewMockClient(crtl)
	mockLogger := LoggerMock.NewMockLogger(crtl)

	// 创建一个mock的AppTable对象
	table := make(map[string]config.Table)
	table["mock-normal-table-id"] = config.Table{
		Name:    "mock-normal-table-name",
		TableID: "mock-normal-table-id",
		ViewID:  "mock-normal-view-id",
	}
	table["mock-table-id"] = config.Table{
		Name:    "mock-table-name",
		TableID: "mock-table-id",
		ViewID:  "mock-view-id",
	}
	mockAppTableConfig := &config.AppTable{
		AppToken: "mock-app-token",
		Tables:   table,
	}

	mockOpenIds := []config.OpenID{
		{
			OpenID: "mock-open-id",
			Name:   "mock-open-id-name",
		},
	}
	mockChatIds := []config.ChatID{
		{
			ChatID: "mock-chat-id",
			Name:   "mock-chat-id-name",
		},
	}

	mockBatchNoticeConfig := &config.BatchNoticeConfig{
		OpenIDs: mockOpenIds,
		ChatIDs: mockChatIds,
		Content: config.Content{
			Type: "mock-type",
			Data: config.Data{
				TemplateID:          "mock-template-id",
				TemplateVersionName: "mock-template-version-name",
				TemplateVariable: config.TemplateVariable{
					FeedbackContent: "mock-feedback-content",
					FeedbackSource:  "mock-feedback-source",
					FeedbackType:    "mock-feedback-type",
				},
			},
		},
	}
	return &Sheet{
		c:    mockClient,
		log:  mockLogger,
		o:    mockAuth,
		cfg:  mockAppTableConfig,
		bcfg: mockBatchNoticeConfig,
	}, mockAuth, mockClient, mockLogger
}

// uc
var uc = ijwt.UserClaims{
	RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), // 这里mock 1小时过期
	},
	TableID:       "mock-table-id",
	NormalTableID: "mock-normal-table-id",
}

func TestCreateApp(t *testing.T) {
	type testCase struct {
		name          string
		ginCtx        *gin.Context
		req           request.CreateAppReq
		uc            ijwt.UserClaims
		setupMocks    func(mockAuth *AuthMock.MockAuthService, mockClient *FeishuMock.MockClient, mockLogger *LoggerMock.MockLogger) // 设置mock期望的函数
		expectedCode  int
		expectedMsg   string
		expectedError bool
	}
	// 表格驱动测试用例
	testCases := []testCase{
		{
			name:   "create app success",
			ginCtx: &gin.Context{},
			req: request.CreateAppReq{
				Name:        "Test_App",
				FolderToken: "test_folder_token",
			},
			uc: uc,
			setupMocks: func(mockAuth *AuthMock.MockAuthService, mockClient *FeishuMock.MockClient, mockLogger *LoggerMock.MockLogger) {
				// 设置成功场景的mock期望
				mockAuth.EXPECT().
					GetAccessToken().
					Return("mock_access_token").
					Times(1)

				// 创建成功的mock响应
				mockSuccessResp := &larkbitable.CreateAppResp{
					CodeError: larkcore.CodeError{
						Code: 0,
					},
					Data: &larkbitable.CreateAppRespData{
						// 只设置测试需要验证的字段
						App: &larkbitable.App{
							AppToken: stringPtr("test_app_token"),
							Name:     stringPtr("Test_App"),
						},
					},
				}

				mockClient.EXPECT().
					CreateAPP(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(mockSuccessResp, nil).
					Times(1)
			},
			expectedCode:  0,
			expectedMsg:   "Success",
			expectedError: false,
		},
		{
			name:   "client error",
			ginCtx: &gin.Context{},
			req: request.CreateAppReq{
				Name:        "Test_App",
				FolderToken: "test_folder_token",
			},
			uc: uc,
			setupMocks: func(mockAuth *AuthMock.MockAuthService, mockClient *FeishuMock.MockClient, mockLogger *LoggerMock.MockLogger) {
				mockAuth.EXPECT().
					GetAccessToken().
					Return("mock_access_token").
					Times(1)

				mockClient.EXPECT().
					CreateAPP(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("network error")).
					Times(1)

				// 设置日志的期望
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).
					Return().Times(1)
			},
			expectedCode:  500,
			expectedMsg:   "Internal Server Error",
			expectedError: true,
		},
	}

	// 执行测试
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置 mock 控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建mock实例
			sheet, mockAuth, mockClient, mockLogger := NewMockSheet(ctrl)

			// 设置当前测试用例的mock期望
			if tc.setupMocks != nil {
				tc.setupMocks(mockAuth, mockClient, mockLogger)
			}

			// 执行被测试的方法
			result, err := sheet.CreateApp(tc.ginCtx, tc.req, tc.uc)

			// 验证结果
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedCode, result.Code)
			assert.Equal(t, tc.expectedMsg, result.Message)

			// 根据测试用例验证Data字段
			if tc.expectedCode == 0 {
				assert.NotNil(t, result.Data) // 成功时应该有数据
			}
		})
	}
}

func TestCopyApp(t *testing.T) {
	type testCase struct {
		name          string
		ginCtx        *gin.Context
		req           request.CopyAppReq
		uc            ijwt.UserClaims
		setupMocks    func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger)
		expectedCode  int
		expectedMsg   string
		expectedError bool
	}
	// 表格驱动测试用例
	testCases := []testCase{
		{
			name:   "copy app success",
			ginCtx: &gin.Context{},
			req: request.CopyAppReq{
				AppToken:       "mock-source-app-token",
				Name:           "Copied_App",
				FolderToken:    "mock-folder-token",
				WithoutContent: true,
				TimeZone:       "Asia/Shanghai",
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {
				// 设置成功场景的mock期望
				auth.EXPECT().GetAccessToken().Return("mock-access-token").Times(1)

				// 创建成功的mock响应

				mockSuccessResp := &larkbitable.CopyAppResp{
					CodeError: larkcore.CodeError{
						Code: 0,
					},
					Data: &larkbitable.CopyAppRespData{
						App: &larkbitable.App{
							AppToken: stringPtr("mock-new-app-token"),
							Name:     stringPtr("Copied_App"),
						},
					},
				}

				client.EXPECT().CopyAPP(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(mockSuccessResp, nil).Times(1)
			},
			expectedCode:  0,
			expectedMsg:   "Success",
			expectedError: false,
		},
		{
			name:   "client error",
			ginCtx: &gin.Context{},
			req: request.CopyAppReq{
				AppToken:       "mock-source-app-token",
				Name:           "Copied_App",
				FolderToken:    "mock-folder-token",
				WithoutContent: true,
				TimeZone:       "Asia/Shanghai",
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, mockLogger *LoggerMock.MockLogger) {
				auth.EXPECT().GetAccessToken().Return("mock-access-token").Times(1)

				client.EXPECT().CopyAPP(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("network error")).Times(1)

				// 设置日志的期望
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).
					Return().Times(1)

			},
			expectedCode:  500,
			expectedMsg:   "Internal Server Error",
			expectedError: true,
		},
	}

	// 执行测试
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置 mock 控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建独立的mock实例
			sheet, mockAuth, mockClient, mockLogger := NewMockSheet(ctrl)

			// 设置当前测试用例的mock期望
			if tc.setupMocks != nil {
				tc.setupMocks(mockAuth, mockClient, mockLogger)
			}

			// 执行被测试的方法
			result, err := sheet.CopyApp(tc.ginCtx, tc.req, tc.uc)

			// 验证结果
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedCode, result.Code)
			assert.Equal(t, tc.expectedMsg, result.Message)

			// 根据测试用例验证Data字段
			if tc.expectedCode == 0 {
				assert.NotNil(t, result.Data) // 成功时应该有数据
			} else {
				assert.Nil(t, result.Data) // 客户端错误时Data为nil
			}
		})
	}
}

func TestCreateAppTableRecord(t *testing.T) {
	type testCase struct {
		name          string
		ginCtx        *gin.Context
		req           request.CreateAppTableRecordReq
		uc            ijwt.UserClaims
		setupMocks    func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger)
		expectedCode  int
		expectedMsg   string
		expectedError bool
	}

	testCases := []testCase{
		{
			name:   "create record success",
			ginCtx: &gin.Context{},
			req: request.CreateAppTableRecordReq{
				Content:                "mock-content",
				ProblemType:            "mock-type",
				IgnoreConsistencyCheck: true,
				Fields: map[string]interface{}{
					"col1": "val1",
				},
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {
				auth.EXPECT().GetAccessToken().Return("mock-access-token").Times(1)

				mockResp := &larkbitable.CreateAppTableRecordResp{
					CodeError: larkcore.CodeError{Code: 0},
					Data: &larkbitable.CreateAppTableRecordRespData{
						Record: &larkbitable.AppTableRecord{
							RecordId: stringPtr("mock-record-id"),
						},
					},
				}

				client.EXPECT().CreateAppTableRecord(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(mockResp, nil).Times(1)
			},
			expectedCode:  0,
			expectedMsg:   "Success",
			expectedError: false,
		},
		{
			name:   "client error",
			ginCtx: &gin.Context{},
			req: request.CreateAppTableRecordReq{
				Content:     "mock-content",
				ProblemType: "mock-type",
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {
				auth.EXPECT().GetAccessToken().Return("mock-access-token").Times(1)

				client.EXPECT().CreateAppTableRecord(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("network error")).Times(1)

				log.EXPECT().Errorf(gomock.Any(), gomock.Any()).
					Return().Times(1)
			},
			expectedCode:  500,
			expectedMsg:   "Internal Server Error",
			expectedError: true,
		},
		{
			name:   "invalid table ID",
			ginCtx: &gin.Context{},
			req:    request.CreateAppTableRecordReq{},
			uc:     ijwt.UserClaims{TableID: "invalid_table"},
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {

			},
			expectedCode:  400,
			expectedMsg:   "Bad Request",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockAuth, mockClient, mockLogger := NewMockSheet(ctrl)
			sheet.Testing = true
			if tc.setupMocks != nil {
				tc.setupMocks(mockAuth, mockClient, mockLogger)
			}

			result, err := sheet.CreateAppTableRecord(tc.ginCtx, tc.req, tc.uc)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedCode, result.Code)
			assert.Equal(t, tc.expectedMsg, result.Message)

			if tc.expectedCode == 0 {
				assert.NotNil(t, result.Data)
			} else {
				assert.Nil(t, result.Data)
			}
		})
	}
}

func TestGetAppTableRecord(t *testing.T) {
	type testCase struct {
		name          string
		ginCtx        *gin.Context
		req           request.GetAppTableRecordReq
		uc            ijwt.UserClaims
		setupMocks    func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger)
		expectedCode  int
		expectedMsg   string
		expectedError bool
	}

	testCases := []testCase{
		{
			name:   "successful record retrieval",
			ginCtx: &gin.Context{},
			req: request.GetAppTableRecordReq{
				PageToken:  "", // 第一次为空，这里模拟第一次
				FieldNames: []string{"name", "age"},
				SortOrders: "created_at",
				Desc:       true,
				FilterName: "status",
				FilterVal:  "active",
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {
				auth.EXPECT().GetAccessToken().Return("valid_token").Times(1)

				// 构建模拟的成功响应
				mockResp := &larkbitable.SearchAppTableRecordResp{
					CodeError: larkcore.CodeError{Code: 0},
					Data: &larkbitable.SearchAppTableRecordRespData{
						Items: []*larkbitable.AppTableRecord{
							{RecordId: stringPtr("rec123")},
							{RecordId: stringPtr("rec456")},
						},
						HasMore: boolPtr(false),
						Total:   intPtr(2),
					},
				}

				client.EXPECT().GetAppTableRecord(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(mockResp, nil).Times(1)
			},
			expectedCode:  0,
			expectedMsg:   "Success",
			expectedError: false,
		},
		{
			name:   "invalid table ID",
			ginCtx: &gin.Context{},
			req:    request.GetAppTableRecordReq{},
			uc:     ijwt.UserClaims{TableID: "invalid_table"},
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {

			},
			expectedCode:  400,
			expectedMsg:   "Bad Request",
			expectedError: true,
		},
		{
			name:   "client error",
			ginCtx: &gin.Context{},
			req: request.GetAppTableRecordReq{
				PageToken:  "page_123",
				FieldNames: []string{"name"},
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {
				auth.EXPECT().GetAccessToken().Return("valid_token").Times(1)
				client.EXPECT().GetAppTableRecord(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("network error")).Times(1)
				log.EXPECT().Errorf(gomock.Any(), gomock.Any()).
					Return().Times(1)
			},
			expectedCode:  500,
			expectedMsg:   "Internal Server Error",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockAuth, mockClient, mockLogger := NewMockSheet(ctrl)
			sheet.Testing = true
			if tc.setupMocks != nil {
				tc.setupMocks(mockAuth, mockClient, mockLogger)
			}

			result, err := sheet.GetAppTableRecord(tc.ginCtx, tc.req, tc.uc)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedCode, result.Code)
			assert.Equal(t, tc.expectedMsg, result.Message)

			if tc.expectedCode == 0 {
				assert.NotNil(t, result.Data)
			} else {
				assert.Nil(t, result.Data)
			}
		})
	}
}

func TestGetNormalRecord(t *testing.T) {
	type testCase struct {
		name          string
		ginCtx        *gin.Context
		req           request.GetAppTableRecordReq
		uc            ijwt.UserClaims
		setupMocks    func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger)
		expectedCode  int
		expectedMsg   string
		expectedError bool
	}

	testCases := []testCase{
		{
			name:   "successful record retrieval",
			ginCtx: &gin.Context{},
			req: request.GetAppTableRecordReq{
				PageToken:  "", // 第一次为空，这里模拟第一次
				FieldNames: []string{"name", "age"},
				SortOrders: "created_at",
				Desc:       true,
				FilterName: "status",
				FilterVal:  "active",
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {
				auth.EXPECT().GetAccessToken().Return("valid_token").Times(1)

				// 构建模拟的成功响应
				mockResp := &larkbitable.SearchAppTableRecordResp{
					CodeError: larkcore.CodeError{Code: 0},
					Data: &larkbitable.SearchAppTableRecordRespData{
						Items: []*larkbitable.AppTableRecord{
							{RecordId: stringPtr("rec123")},
							{RecordId: stringPtr("rec456")},
						},
						HasMore: boolPtr(false),
						Total:   intPtr(2),
					},
				}

				client.EXPECT().GetAppTableRecord(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(mockResp, nil).Times(1)
			},
			expectedCode:  0,
			expectedMsg:   "Success",
			expectedError: false,
		},
		{
			name:   "invalid table ID",
			ginCtx: &gin.Context{},
			req:    request.GetAppTableRecordReq{},
			uc:     ijwt.UserClaims{TableID: "invalid_table"},
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {

			},
			expectedCode:  400,
			expectedMsg:   "Bad Request",
			expectedError: true,
		},
		{
			name:   "client error",
			ginCtx: &gin.Context{},
			req: request.GetAppTableRecordReq{
				PageToken:  "page_123",
				FieldNames: []string{"name"},
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {
				auth.EXPECT().GetAccessToken().Return("valid_token").Times(1)
				client.EXPECT().GetAppTableRecord(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("network error")).Times(1)
				log.EXPECT().Errorf(gomock.Any(), gomock.Any()).
					Return().Times(1)
			},
			expectedCode:  500,
			expectedMsg:   "Internal Server Error",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockAuth, mockClient, mockLogger := NewMockSheet(ctrl)
			sheet.Testing = true
			if tc.setupMocks != nil {
				tc.setupMocks(mockAuth, mockClient, mockLogger)
			}

			result, err := sheet.GetNormalRecord(tc.ginCtx, tc.req, tc.uc)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedCode, result.Code)
			assert.Equal(t, tc.expectedMsg, result.Message)

			if tc.expectedCode == 0 {
				assert.NotNil(t, result.Data)
			} else {
				assert.Nil(t, result.Data)
			}
		})
	}
}

func TestGetPhotoUrl(t *testing.T) {
	type testCase struct {
		name          string
		ginCtx        *gin.Context
		req           request.GetPhotoUrlReq
		uc            ijwt.UserClaims
		setupMocks    func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger)
		expectedCode  int
		expectedMsg   string
		expectedError bool
	}

	testCases := []testCase{
		{
			name:   "successful get photo url",
			ginCtx: &gin.Context{},
			req: request.GetPhotoUrlReq{
				FileTokens: []string{"file_token1", "file_token2"},
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {
				auth.EXPECT().GetAccessToken().Return("mock_access_token").Times(1)

				// 构建模拟的成功响应
				mockResp := &larkdrive.BatchGetTmpDownloadUrlMediaResp{
					CodeError: larkcore.CodeError{Code: 0},
					Data: &larkdrive.BatchGetTmpDownloadUrlMediaRespData{
						TmpDownloadUrls: []*larkdrive.TmpDownloadUrl{
							{
								FileToken:      stringPtr("file_token1"),
								TmpDownloadUrl: stringPtr("https://example.com/file1.jpg"),
							},
							{
								FileToken:      stringPtr("file_token2"),
								TmpDownloadUrl: stringPtr("https://example.com/file2.jpg"),
							},
						},
					},
				}

				client.EXPECT().GetPhotoUrl(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(mockResp, nil).Times(1)
			},
			expectedCode:  0,
			expectedMsg:   "Success",
			expectedError: false,
		},
		{
			name:   "client error",
			ginCtx: &gin.Context{},
			req: request.GetPhotoUrlReq{
				FileTokens: []string{"file_token1"},
			},
			uc: uc,
			setupMocks: func(auth *AuthMock.MockAuthService, client *FeishuMock.MockClient, log *LoggerMock.MockLogger) {
				auth.EXPECT().GetAccessToken().Return("mock_access_token").Times(1)

				// 模拟API调用返回错误
				client.EXPECT().GetPhotoUrl(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("network error")).Times(1)

				log.EXPECT().Errorf(gomock.Any(), gomock.Any()).
					Return().Times(1)
			},
			expectedCode:  500,
			expectedMsg:   "Internal Server Error",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockAuth, mockClient, mockLogger := NewMockSheet(ctrl)
			sheet.Testing = true
			if tc.setupMocks != nil {
				tc.setupMocks(mockAuth, mockClient, mockLogger)
			}

			result, err := sheet.GetPhotoUrl(tc.ginCtx, tc.req, tc.uc)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedCode, result.Code)
			assert.Equal(t, tc.expectedMsg, result.Message)

			if tc.expectedCode == 0 {
				assert.NotNil(t, result.Data)
			} else {
				assert.Nil(t, result.Data)
			}
		})
	}
}

func TestIsEmptyValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"empty slice", []int{}, true},
		{"non-empty slice", []int{1}, false},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
		{"nil pointer", (*int)(nil), true},
		{"non-nil pointer", func() *int { i := 1; return &i }(), false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
		{"false bool", false, true},
		{"true bool", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			assert.Equal(t, tt.expected, isEmptyValue(v))
		})
	}
}

func TestFillFields(t *testing.T) {
	tests := []struct {
		name          string
		req           *request.CreateAppTableRecordReq
		expectKeys    []string
		notExpectKeys []string
	}{
		{
			name: "all required fields filled, optional empty",
			req: &request.CreateAppTableRecordReq{
				Fields:    make(map[string]interface{}),
				StudentID: "12345",
				Contact:   "test@example.com",
				Content:   "反馈内容",
			},
			expectKeys:    []string{"用户ID", "联系方式（QQ/邮箱）", "反馈内容", "提交时间", "问题状态"},
			notExpectKeys: []string{"问题类型", "问题来源", "截图"},
		},
		{
			name: "optional fields filled",
			req: &request.CreateAppTableRecordReq{
				Fields:        make(map[string]interface{}),
				StudentID:     "12345",
				Contact:       "test@example.com",
				Content:       "反馈内容",
				ProblemType:   "Bug",
				ProblemSource: "Web",
			},
			expectKeys:    []string{"用户ID", "联系方式（QQ/邮箱）", "反馈内容", "问题类型", "问题来源", "提交时间", "问题状态"},
			notExpectKeys: []string{"截图"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fillFields(tt.req)

			// 自动填充校验
			assert.Equal(t, "处理中", tt.req.Status)
			assert.NotZero(t, tt.req.SubmitTIme)
			// 提交时间应为当前时间附近（5秒误差）
			assert.InDelta(t, time.Now().UnixMilli(), tt.req.SubmitTIme, 5000)

			// 校验应包含的字段
			for _, key := range tt.expectKeys {
				_, ok := tt.req.Fields[key]
				assert.Truef(t, ok, "Fields should contain key: %s", key)
			}

			// 校验不应包含的字段
			for _, key := range tt.notExpectKeys {
				_, ok := tt.req.Fields[key]
				assert.Falsef(t, ok, "Fields should NOT contain key: %s", key)
			}
		})
	}
}

// 辅助函数：创建字符串指针
func stringPtr(s string) *string {
	return &s
}
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
