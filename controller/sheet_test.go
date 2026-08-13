package controller

import (
	"testing"
	"time"

	"github.com/muxi-Infra/FeedBack-Backend/api/request/v1"
	"github.com/muxi-Infra/FeedBack-Backend/domain"

	"github.com/muxi-Infra/FeedBack-Backend/pkg/ijwt"
	ServiceMock "github.com/muxi-Infra/FeedBack-Backend/service/mock"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// 创建一个 mock 的 SheetV1 对象
func NewMockSheet(crtl *gomock.Controller) (*SheetV1, *ServiceMock.MockSheetService, *ServiceMock.MockMessageService) {
	// 接口 mock
	mockSheetService := ServiceMock.NewMockSheetService(crtl)
	mockMessageService := ServiceMock.NewMockMessageService(crtl)

	return &SheetV1{
		s: mockSheetService,
		m: mockMessageService,
	}, mockSheetService, mockMessageService
}

// uc
var uc = ijwt.UserClaims{
	RegisteredClaims: jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), // 这里mock 1小时过期
	},
	TableIdentity: "mock-table-identity",
	TableName:     "mock-table-name",
	TableToken:    "mock-table-token",
	TableId:       "mock-table-id",
	ViewId:        "mock-view-id",
}

func TestCreateAppTableRecord(t *testing.T) {
	type testCase struct {
		name          string
		req           v1.CreatTableRecordReg
		uc            ijwt.UserClaims
		setupMocks    func(mockSheetSvc *ServiceMock.MockSheetService, mockMessageSvc *ServiceMock.MockMessageService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		{
			name: "create record success",
			req: v1.CreatTableRecordReg{
				TableIdentify: stringPtr("mock-table-identity"),
				StudentID:     stringPtr("2021001234"),
				Content:       stringPtr("测试反馈内容"),
				Images:        []string{"token1", "token2"},
				ContactInfo:   stringPtr("test@example.com"),
				ExtraRecord: map[string]interface{}{
					"额外字段": "额外值",
				},
			},
			uc: uc,
			setupMocks: func(mockSheetSvc *ServiceMock.MockSheetService, mockMessageSvc *ServiceMock.MockMessageService) {
				mockSheetSvc.EXPECT().
					CreateLarkRecord(gomock.Any(), gomock.Any()).
					Return(stringPtr("mock-record-id"), nil)

				// Allow CreateDBRecord called by background goroutine
				mockSheetSvc.EXPECT().
					CreateDBRecord(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				// Mock the goroutine calls
				mockSheetSvc.EXPECT().
					GetTableRecordReqByRecordID(gomock.Any(), gomock.Any()).
					Return(map[string]any{}, stringPtr("http://mock-url.com"), nil).
					AnyTimes()

				mockMessageSvc.EXPECT().
					SendLarkNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()
			},
			expectedCode:  0,
			expectedError: false,
		},
		{
			name: "create record with missing student id",
			req: v1.CreatTableRecordReg{
				TableIdentify: stringPtr("mock-table-identity"),
				StudentID:     nil,
				Content:       stringPtr("测试反馈内容"),
			},
			uc:            uc,
			setupMocks:    nil,
			expectedCode:  0,
			expectedError: true,
		},
		{
			name: "create record with invalid student id length",
			req: v1.CreatTableRecordReg{
				TableIdentify: stringPtr("mock-table-identity"),
				StudentID:     stringPtr("123"),
				Content:       stringPtr("测试反馈内容"),
			},
			uc:            uc,
			setupMocks:    nil,
			expectedCode:  0,
			expectedError: true,
		},
		{
			name: "create record with missing content",
			req: v1.CreatTableRecordReg{
				TableIdentify: stringPtr("mock-table-identity"),
				StudentID:     stringPtr("2021001234"),
				Content:       nil,
			},
			uc:            uc,
			setupMocks:    nil,
			expectedCode:  0,
			expectedError: true,
		},
		{
			name: "create record with empty content",
			req: v1.CreatTableRecordReg{
				TableIdentify: stringPtr("mock-table-identity"),
				StudentID:     stringPtr("2021001234"),
				Content:       stringPtr(""),
			},
			uc:            uc,
			setupMocks:    nil,
			expectedCode:  0,
			expectedError: true,
		},
		{
			name: "create record with table identify mismatch",
			req: v1.CreatTableRecordReg{
				TableIdentify: stringPtr("wrong-table-identity"),
				StudentID:     stringPtr("2021001234"),
				Content:       stringPtr("测试反馈内容"),
			},
			uc:            uc,
			setupMocks:    nil,
			expectedCode:  0,
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			sheet, mockSheetSvc, mockMessageSvc := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSheetSvc, mockMessageSvc)
			}

			result, err := sheet.CreateTableRecord(&gin.Context{}, tc.req, tc.uc)

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
		req           v1.GetTableRecordReq
		uc            ijwt.UserClaims
		setupMocks    func(mockSheetSvc *ServiceMock.MockSheetService, mockMessageSvc *ServiceMock.MockMessageService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		{
			name: "get record success",
			req: v1.GetTableRecordReq{
				TableIdentify: stringPtr("mock-table-identity"),
				KeyFieldName:  stringPtr("mock-name"),
				KeyFieldValue: stringPtr("mock-value"),
				RecordNames:   []string{"field1", "field2"},
			},
			uc: uc,
			setupMocks: func(mockSheetSvc *ServiceMock.MockSheetService, mockMessageSvc *ServiceMock.MockMessageService) {
				mockSheetSvc.EXPECT().
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

			sheet, mockSheetSvc, mockMessageSvc := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSheetSvc, mockMessageSvc)
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

func TestGetFAQResolutionRecord(t *testing.T) {
	type testCase struct {
		name          string
		req           v1.GetFAQProblemTableRecordReg
		uc            ijwt.UserClaims
		setupMocks    func(mockSheetSvc *ServiceMock.MockSheetService, mockMessageSvc *ServiceMock.MockMessageService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		{
			name: "get FAQ record success",
			req: v1.GetFAQProblemTableRecordReg{
				TableIdentify: stringPtr("mock-table-identity"),
				StudentID:     stringPtr("mock-student-id"),
				RecordNames:   []string{"mock-name-1"},
			},
			uc: uc,
			setupMocks: func(mockSheetSvc *ServiceMock.MockSheetService, mockMessageSvc *ServiceMock.MockMessageService) {
				mockSheetSvc.EXPECT().
					GetFAQProblemTableRecord(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&domain.FAQTableRecords{
						Records: []domain.FAQTableRecord{},
						Total:   intPtr(0),
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

			sheet, mockSheetSvc, mockMessageSvc := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSheetSvc, mockMessageSvc)
			}

			result, err := sheet.GetFAQResolutionRecord(&gin.Context{}, tc.req, tc.uc)

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
		req           v1.GetPhotoUrlReq
		setupMocks    func(mockSheetSvc *ServiceMock.MockSheetService, mockMessageSvc *ServiceMock.MockMessageService)
		expectedCode  int
		expectedError bool
	}

	testCases := []testCase{
		{
			name: "get photo url success",
			req: v1.GetPhotoUrlReq{
				FileTokens: []string{"token1", "token2"},
			},
			setupMocks: func(mockSheetSvc *ServiceMock.MockSheetService, mockMessageSvc *ServiceMock.MockMessageService) {
				mockSheetSvc.EXPECT().
					GetPhotoUrl(gomock.Any()).
					Return([]domain.File{
						{
							FileToken:      stringPtr("token1"),
							TmpDownloadURL: stringPtr("https://example.com/token1"),
						},
						{
							FileToken:      stringPtr("token2"),
							TmpDownloadURL: stringPtr("https://example.com/token2"),
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

			sheet, mockSheetSvc, mockMessageSvc := NewMockSheet(ctrl)

			if tc.setupMocks != nil {
				tc.setupMocks(mockSheetSvc, mockMessageSvc)
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
