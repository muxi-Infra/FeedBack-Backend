package controller

import (
	"testing"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/config"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	LoggerMock "github.com/muxi-Infra/FeedBack-Backend/pkg/logger/mock"
	ServiceMock "github.com/muxi-Infra/FeedBack-Backend/service/mock"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/stretchr/testify/assert"
)

// 创建一个 mock 的 Sheet 对象
func NewMockSheet(crtl *gomock.Controller) (*Sheet, *ServiceMock.MockSheetService, *LoggerMock.MockLogger) {
	// 接口 mock
	mockSheetService := ServiceMock.NewMockSheetService(crtl)
	mockLogger := LoggerMock.NewMockLogger(crtl)

	// 创建一个 mock 的 AppTable 对象
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

	return &Sheet{
		log: mockLogger,
		cfg: mockAppTableConfig,
		s:   mockSheetService,
	}, mockSheetService, mockLogger
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
		req           request.CreateAppReq
		setupMocks    func(mockSvc *ServiceMock.MockSheetService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		{
			name: "create app success",
			req: request.CreateAppReq{
				Name:        "Test_App",
				FolderToken: "test_folder_token",
			},
			setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
				mockSvc.EXPECT().
					CreateAPP(gomock.Any(), gomock.Any()).
					Return(&larkbitable.CreateAppResp{
						Data: &larkbitable.CreateAppRespData{
							App: &larkbitable.App{
								AppToken: stringPtr("test_app_token"),
								Name:     stringPtr("Test_App"),
							},
						},
					}, nil)
			},
			expectedCode:  0,
			expectedError: false,
		},
		//{
		//	name: "service error",
		//	req: request.CreateAppReq{
		//		Name:        "Test_App",
		//		FolderToken: "test_folder_token",
		//	},
		//	setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
		//		mockSvc.EXPECT().
		//			CreateAPP(gomock.Any(), gomock.Any()).
		//			Return(nil, errors.New("service error"))
		//	},
		//	expectedCode:  500,
		//	expectedError: true,
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockSvc, _ := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSvc)
			}

			result, err := sheet.CreateApp(&gin.Context{}, tc.req, uc)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedCode, result.Code)

			if tc.expectedCode == 0 {
				assert.NotNil(t, result.Data)
			}
		})
	}
}

func TestCopyApp(t *testing.T) {
	type testCase struct {
		name          string
		req           request.CopyAppReq
		setupMocks    func(mockSvc *ServiceMock.MockSheetService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		{
			name: "copy app success",
			req: request.CopyAppReq{
				AppToken:       "mock-app-token",
				Name:           "Copied_App",
				FolderToken:    "mock-folder-token",
				WithoutContent: false,
				TimeZone:       "UTC",
			},
			setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
				mockSvc.EXPECT().
					CopyAPP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&larkbitable.CopyAppResp{
						Data: &larkbitable.CopyAppRespData{
							App: &larkbitable.App{
								AppToken: stringPtr("copied_app_token"),
								Name:     stringPtr("Copied_App"),
							},
						},
					}, nil)
			},
			expectedCode:  0,
			expectedError: false,
		},
		//{
		//	name: "service error",
		//	req: request.CopyAppReq{
		//		AppToken:       "mock-app-token",
		//		Name:           "Copied_App",
		//		FolderToken:    "mock-folder-token",
		//		WithoutContent: true,
		//		TimeZone:       "UTC",
		//	},
		//	setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
		//		mockSvc.EXPECT().
		//			CopyAPP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		//			Return(nil, errors.New("service error"))
		//	},
		//	expectedCode:  500,
		//	expectedError: true,
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockSvc, _ := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSvc)
			}

			result, err := sheet.CopyApp(&gin.Context{}, tc.req, uc)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.Nil(t, result.Data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.NotNil(t, result.Data)
			}
		})
	}
}
func TestCreateAppTableRecord(t *testing.T) {
	type testCase struct {
		name          string
		req           request.CreateAppTableRecordReq
		uc            ijwt.UserClaims
		setupMocks    func(mockSvc *ServiceMock.MockSheetService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		//{
		//	name: "table not found",
		//	req: request.CreateAppTableRecordReq{
		//		Fields: map[string]interface{}{
		//			"name": "test",
		//		},
		//	},
		//	uc: ijwt.UserClaims{
		//		TableID: "not-exist-table-id",
		//	},
		//	setupMocks:    nil,
		//	expectedCode:  400,
		//	expectedError: true,
		//},
		//{
		//	name: "service error",
		//	req: request.CreateAppTableRecordReq{
		//		Fields: map[string]interface{}{
		//			"name": "test",
		//		},
		//	},
		//	uc: uc,
		//	setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
		//		mockSvc.EXPECT().
		//			CreateRecord(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		//			Return(nil, errors.New("service error"))
		//	},
		//	expectedCode:  500,
		//	expectedError: true,
		//},
		{
			name: "create record success",
			req: request.CreateAppTableRecordReq{
				Fields: map[string]interface{}{
					"name": "test",
				},
			},
			uc: uc,
			setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
				mockSvc.EXPECT().
					CreateRecord(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&larkbitable.CreateAppTableRecordResp{
						Data: &larkbitable.CreateAppTableRecordRespData{
							Record: &larkbitable.AppTableRecord{
								RecordId: stringPtr("record-id"),
							},
						},
					}, nil)
			},
			expectedCode:  0,
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockSvc, _ := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSvc)
			}

			result, err := sheet.CreateAppTableRecord(&gin.Context{}, tc.req, tc.uc)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.Nil(t, result.Data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.NotNil(t, result.Data)
			}
		})
	}
}

func TestGetAppTableRecord(t *testing.T) {
	type testCase struct {
		name          string
		req           request.GetAppTableRecordReq
		uc            ijwt.UserClaims
		setupMocks    func(mockSvc *ServiceMock.MockSheetService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		//{
		//	name: "table not found",
		//	req: request.GetAppTableRecordReq{
		//		FieldNames: []string{"name", "age"},
		//		FilterName: "name",
		//		FilterVal:  "test",
		//	},
		//	uc: ijwt.UserClaims{
		//		TableID: "not-exist-table-id",
		//	},
		//	setupMocks:    nil, // 不会调用 service
		//	expectedCode:  400,
		//	expectedError: true,
		//},
		//{
		//	name: "service error",
		//	req: request.GetAppTableRecordReq{
		//		FieldNames: []string{"name"},
		//		FilterName: "name",
		//		FilterVal:  "test",
		//	},
		//	uc: uc, // 正常的 table id
		//	setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
		//		mockSvc.EXPECT().
		//			GetRecord(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		//			Return(nil, errors.New("service error"))
		//	},
		//	expectedCode:  500,
		//	expectedError: true,
		//},
		{
			name: "get record success",
			req: request.GetAppTableRecordReq{
				FieldNames: []string{"name", "age"},
				FilterName: "name",
				FilterVal:  "test",
			},
			uc: uc,
			setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
				mockSvc.EXPECT().
					GetRecord(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&larkbitable.SearchAppTableRecordResp{
						Data: &larkbitable.SearchAppTableRecordRespData{
							Total: intPtr(0),
						},
					}, nil)
			},
			expectedCode:  0,
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockSvc, _ := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSvc)
			}

			result, err := sheet.GetAppTableRecord(&gin.Context{}, tc.req, tc.uc)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.Nil(t, result.Data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.NotNil(t, result.Data)
			}
		})
	}
}

func TestGetNormalRecord(t *testing.T) {
	type testCase struct {
		name          string
		req           request.GetAppTableRecordReq
		uc            ijwt.UserClaims
		setupMocks    func(mockSvc *ServiceMock.MockSheetService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		//{
		//	name: "normal table not found",
		//	req: request.GetAppTableRecordReq{
		//		FieldNames: []string{"name", "age"},
		//		FilterName: "name",
		//		FilterVal:  "test",
		//	},
		//	uc: ijwt.UserClaims{
		//		NormalTableID: "not-exist-normal-table",
		//	},
		//	setupMocks:    nil, // 不会调用 service
		//	expectedCode:  400,
		//	expectedError: true,
		//},
		//{
		//	name: "service error",
		//	req: request.GetAppTableRecordReq{
		//		FieldNames: []string{"name"},
		//		FilterName: "name",
		//		FilterVal:  "test",
		//	},
		//	uc: uc, // 使用已有正常 uc
		//	setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
		//		mockSvc.EXPECT().
		//			GetNormalRecord(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		//			Return(nil, errors.New("service error")).
		//			AnyTimes()
		//	},
		//	expectedCode:  500,
		//	expectedError: true,
		//},
		{
			name: "get normal record success",
			req: request.GetAppTableRecordReq{
				FieldNames: []string{"name", "age"},
				FilterName: "name",
				FilterVal:  "test",
				StudentID:  "stu-123",
			},
			uc: uc,
			setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
				mockSvc.EXPECT().
					GetNormalRecord(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&larkbitable.SearchAppTableRecordResp{
						Data: &larkbitable.SearchAppTableRecordRespData{
							Total: intPtr(0),
						},
					}, nil).
					AnyTimes()
			},
			expectedCode:  0,
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockSvc, _ := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSvc)
			}

			result, err := sheet.GetNormalRecord(&gin.Context{}, tc.req, tc.uc)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.Nil(t, result.Data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.NotNil(t, result.Data)
			}
		})
	}
}

func TestGetPhotoUrl(t *testing.T) {
	type testCase struct {
		name          string
		req           request.GetPhotoUrlReq
		setupMocks    func(mockSvc *ServiceMock.MockSheetService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		//{
		//	name: "service error",
		//	req: request.GetPhotoUrlReq{
		//		FileTokens: []string{"token1", "token2"},
		//	},
		//	setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
		//		mockSvc.EXPECT().
		//			GetPhotoUrl(gomock.Any()).
		//			Return(nil, errors.New("service error")).
		//			AnyTimes()
		//	},
		//	expectedCode:  500,
		//	expectedError: true,
		//},
		{
			name: "get photo url success",
			req: request.GetPhotoUrlReq{
				FileTokens: []string{"token1", "token2"},
			},
			setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
				mockSvc.EXPECT().
					GetPhotoUrl(gomock.Any()).
					Return(&larkdrive.BatchGetTmpDownloadUrlMediaResp{
						Data: &larkdrive.BatchGetTmpDownloadUrlMediaRespData{
							TmpDownloadUrls: []*larkdrive.TmpDownloadUrl{
								{
									FileToken:      stringPtr("token1"),
									TmpDownloadUrl: stringPtr("https://example.com/token1"),
								},
								{
									FileToken:      stringPtr("token2"),
									TmpDownloadUrl: stringPtr("https://example.com/token2"),
								},
							},
						},
					}, nil)
			},
			expectedCode:  0,
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockSvc, _ := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSvc)
			}

			result, err := sheet.GetPhotoUrl(&gin.Context{}, tc.req, uc)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.Nil(t, result.Data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCode, result.Code)
				assert.NotNil(t, result.Data)
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
