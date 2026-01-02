package controller

import (
	"testing"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/domain"

	"github.com/muxi-Infra/FeedBack-Backend/api/request"
	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	LoggerMock "github.com/muxi-Infra/FeedBack-Backend/pkg/logger/mock"
	ServiceMock "github.com/muxi-Infra/FeedBack-Backend/service/mock"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
	"github.com/stretchr/testify/assert"
)

// 创建一个 mock 的 Sheet 对象
func NewMockSheet(crtl *gomock.Controller) (*Sheet, *ServiceMock.MockSheetService, *LoggerMock.MockLogger) {
	// 接口 mock
	mockSheetService := ServiceMock.NewMockSheetService(crtl)
	mockLogger := LoggerMock.NewMockLogger(crtl)

	return &Sheet{
		log: mockLogger,

		s: mockSheetService,
	}, mockSheetService, mockLogger
}

// uc
var uc = ijwt.UserClaims{
	RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), // 这里mock 1小时过期
	},
	TableIdentity: "mock-table-identity",
}

func TestCreateAppTableRecord(t *testing.T) {
	type testCase struct {
		name          string
		req           request.CreatTableRecordReg
		uc            ijwt.UserClaims
		setupMocks    func(mockSvc *ServiceMock.MockSheetService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		//{
		//	name: "table not found",
		//	req: request.CreatTableRecordReg{
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
		//	req: request.CreatTableRecordReg{
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
			req: request.CreatTableRecordReg{
				Record: map[string]interface{}{
					"name": "mock-test",
				},
			},
			uc: uc,
			setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
				mockSvc.EXPECT().
					CreateRecord(gomock.Any(), gomock.Any()).
					Return(stringPtr("mock-test-res"), nil)
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

			result, err := sheet.CreatTableRecord(&gin.Context{}, tc.req, tc.uc)

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

func TestGetTableRecordReqByKey(t *testing.T) {
	type testCase struct {
		name          string
		req           request.GetTableRecordReq
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
		//			GetRecordByStudentID(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		//			Return(nil, errors.New("service error"))
		//	},
		//	expectedCode:  500,
		//	expectedError: true,
		//},
		{
			name: "get record success",
			req: request.GetTableRecordReq{
				KeyFieldName: "mock-name",
			},
			uc: uc,
			setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
				mockSvc.EXPECT().
					GetTableRecordReqByKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&domain.TableRecords{
						HasMore: boolPtr(false),
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

			result, err := sheet.GetTableRecordReqByKey(&gin.Context{}, tc.req, tc.uc)

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

func TestGetNormalProblemTableRecord(t *testing.T) {
	type testCase struct {
		name          string
		req           request.GetNormalProblemTableRecordReg
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
			req: request.GetNormalProblemTableRecordReg{
				RecordNames: []string{"mock-name-1"},
			},
			uc: uc,
			setupMocks: func(mockSvc *ServiceMock.MockSheetService) {
				mockSvc.EXPECT().
					GetNormalProblemTableRecord(gomock.Any(), gomock.Any()).
					Return(&domain.TableRecords{HasMore: boolPtr(false)}, nil).
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

			result, err := sheet.GetNormalProblemTableRecord(&gin.Context{}, tc.req, tc.uc)

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

func boolPtr(b bool) *bool {
	return &b
}
