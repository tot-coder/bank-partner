package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"
	"array-assessment/internal/repositories/repository_mocks"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

func TestAdminHandler(t *testing.T) {
	suite.Run(t, new(AdminHandlerSuite))
}

type AdminHandlerSuite struct {
	suite.Suite
	handler   *AdminHandler
	userRepo  *repository_mocks.MockUserRepositoryInterface
	auditRepo *repository_mocks.MockAuditLogRepositoryInterface
	e         *echo.Echo
}

func (s *AdminHandlerSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.userRepo = repository_mocks.NewMockUserRepositoryInterface(ctrl)
	s.auditRepo = repository_mocks.NewMockAuditLogRepositoryInterface(ctrl)
	s.handler = NewAdminHandler(s.userRepo, s.auditRepo)
	s.e = echo.New()
}

func (s *AdminHandlerSuite) TearDownTest() {
	// gomock assertions happen here
}

func (s *AdminHandlerSuite) createTestUser(role string) *models.User {
	user := &models.User{
		ID:           uuid.New(),
		Email:        fmt.Sprintf("test_%s@example.com", uuid.New().String()),
		FirstName:    "Test",
		LastName:     "User",
		Role:         role,
		PasswordHash: "hashedpassword123",
	}
	return user
}

func (s *AdminHandlerSuite) TestUnlockUser() {
	// Create a locked user
	lockedUser := s.createTestUser(models.RoleCustomer)
	lockedUser.FailedLoginAttempts = 3

	// Create admin user
	adminUser := s.createTestUser(models.RoleAdmin)

	tests := []struct {
		name           string
		userID         string
		contextUserID  uuid.UUID
		expectedStatus int
		expectedError  string
		setupMocks     func()
	}{
		{
			name:           "successful unlock",
			userID:         lockedUser.ID.String(),
			contextUserID:  adminUser.ID,
			expectedStatus: http.StatusOK,
			setupMocks: func() {
				s.userRepo.EXPECT().GetByID(lockedUser.ID).Return(lockedUser, nil).Times(1)
				s.userRepo.EXPECT().UnlockAccount(lockedUser.ID).Return(nil).Times(1)
				s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-uuid",
			contextUserID:  adminUser.ID,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "UUID",
			setupMocks:     func() {}, // No mocks needed for validation error
		},
		{
			name:           "user not found",
			userID:         uuid.New().String(),
			contextUserID:  adminUser.ID,
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
			setupMocks: func() {
				s.userRepo.EXPECT().GetByID(gomock.Any()).Return(nil, repositories.ErrUserNotFound).Times(1)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup mocks for this test case
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()
			s.userRepo = repository_mocks.NewMockUserRepositoryInterface(ctrl)
			s.auditRepo = repository_mocks.NewMockAuditLogRepositoryInterface(ctrl)
			s.handler = NewAdminHandler(s.userRepo, s.auditRepo)

			tt.setupMocks()

			// Setup request
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%s/unlock", tt.userID), nil)
			rec := httptest.NewRecorder()
			c := s.e.NewContext(req, rec)
			c.SetParamNames("userId")
			c.SetParamValues(tt.userID)
			c.Set("user_id", tt.contextUserID)

			// Execute
			err := s.handler.UnlockUser(c)

			// Assert
			if tt.expectedError != "" {
				s.NoError(err) // SendError returns nil, error is in response body
				s.Equal(tt.expectedStatus, rec.Code)

				// Parse and verify error response
				var errorResp ErrorResponse
				parseErr := json.Unmarshal(rec.Body.Bytes(), &errorResp)
				s.NoError(parseErr)

				// Check both message and details for the expected error
				found := strings.Contains(errorResp.Error.Message, tt.expectedError)
				if !found && len(errorResp.Error.Details) > 0 {
					for _, detail := range errorResp.Error.Details {
						if strings.Contains(detail, tt.expectedError) {
							found = true
							break
						}
					}
				}
				s.True(found, "Expected error '%s' not found in message or details", tt.expectedError)
			} else {
				s.NoError(err)
				s.Equal(tt.expectedStatus, rec.Code)
			}
		})
	}
}

func (s *AdminHandlerSuite) TestListUsers() {
	tests := []struct {
		name           string
		queryParams    map[string]string
		expectedStatus int
		setupMocks     func() []*models.User
	}{
		{
			name:           "successful list with defaults",
			queryParams:    map[string]string{},
			expectedStatus: http.StatusOK,
			setupMocks: func() []*models.User {
				users := []*models.User{
					s.createTestUser(models.RoleCustomer),
					s.createTestUser(models.RoleCustomer),
					s.createTestUser(models.RoleCustomer),
					s.createTestUser(models.RoleCustomer),
					s.createTestUser(models.RoleCustomer),
					s.createTestUser(models.RoleAdmin),
				}
				s.userRepo.EXPECT().ListUsers(0, 20).Return(users, int64(len(users)), nil).Times(1)
				return users
			},
		},
		{
			name: "successful list with pagination",
			queryParams: map[string]string{
				"page":  "1",
				"limit": "3",
			},
			expectedStatus: http.StatusOK,
			setupMocks: func() []*models.User {
				users := []*models.User{
					s.createTestUser(models.RoleCustomer),
					s.createTestUser(models.RoleCustomer),
					s.createTestUser(models.RoleCustomer),
				}
				s.userRepo.EXPECT().ListUsers(0, 3).Return(users, int64(len(users)), nil).Times(1)
				return users
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup mocks for this test case
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()
			s.userRepo = repository_mocks.NewMockUserRepositoryInterface(ctrl)
			s.auditRepo = repository_mocks.NewMockAuditLogRepositoryInterface(ctrl)
			s.handler = NewAdminHandler(s.userRepo, s.auditRepo)

			expectedUsers := tt.setupMocks()

			// Setup request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()
			c := s.e.NewContext(req, rec)

			// Execute
			err := s.handler.ListUsers(c)

			// Assert
			s.NoError(err)
			s.Equal(tt.expectedStatus, rec.Code)

			var response SuccessResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			s.NoError(err)

			users, ok := response.Data.([]interface{})
			s.True(ok)
			s.Equal(len(expectedUsers), len(users))
		})
	}
}

func (s *AdminHandlerSuite) TestGetUserByID() {
	// Create test user
	testUser := s.createTestUser(models.RoleCustomer)
	adminUser := s.createTestUser(models.RoleAdmin)

	tests := []struct {
		name           string
		userID         string
		contextUserID  uuid.UUID
		expectedStatus int
		expectedError  string
		setupMocks     func()
	}{
		{
			name:           "successful get",
			userID:         testUser.ID.String(),
			contextUserID:  adminUser.ID,
			expectedStatus: http.StatusOK,
			setupMocks: func() {
				s.userRepo.EXPECT().GetByID(testUser.ID).Return(testUser, nil).Times(1)
			},
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-uuid",
			contextUserID:  adminUser.ID,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "UUID",
			setupMocks:     func() {}, // No mocks needed for validation error
		},
		{
			name:           "user not found",
			userID:         uuid.New().String(),
			contextUserID:  adminUser.ID,
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
			setupMocks: func() {
				s.userRepo.EXPECT().GetByID(gomock.Any()).Return(nil, repositories.ErrUserNotFound).Times(1)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup mocks for this test case
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()
			s.userRepo = repository_mocks.NewMockUserRepositoryInterface(ctrl)
			s.auditRepo = repository_mocks.NewMockAuditLogRepositoryInterface(ctrl)
			s.handler = NewAdminHandler(s.userRepo, s.auditRepo)

			tt.setupMocks()

			// Setup request
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/users/%s", tt.userID), nil)
			rec := httptest.NewRecorder()
			c := s.e.NewContext(req, rec)
			c.SetParamNames("userId")
			c.SetParamValues(tt.userID)
			c.Set("user_id", tt.contextUserID)

			// Execute
			err := s.handler.GetUserByID(c)

			// Assert
			if tt.expectedError != "" {
				s.NoError(err) // SendError returns nil, error is in response body
				s.Equal(tt.expectedStatus, rec.Code)

				// Parse and verify error response
				var errorResp ErrorResponse
				parseErr := json.Unmarshal(rec.Body.Bytes(), &errorResp)
				s.NoError(parseErr)

				// Check both message and details for the expected error
				found := strings.Contains(errorResp.Error.Message, tt.expectedError)
				if !found && len(errorResp.Error.Details) > 0 {
					for _, detail := range errorResp.Error.Details {
						if strings.Contains(detail, tt.expectedError) {
							found = true
							break
						}
					}
				}
				s.True(found, "Expected error '%s' not found in message or details", tt.expectedError)
			} else {
				s.NoError(err)
				s.Equal(tt.expectedStatus, rec.Code)

				var response SuccessResponse
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				s.NoError(err)

				userData := response.Data.(map[string]interface{})
				s.Equal(testUser.ID.String(), userData["id"])
				s.Equal(testUser.Email, userData["email"])
			}
		})
	}
}

func (s *AdminHandlerSuite) TestDeleteUser() {
	// Create test users
	userToDelete := s.createTestUser(models.RoleCustomer)
	adminUser := s.createTestUser(models.RoleAdmin)

	tests := []struct {
		name           string
		userID         string
		contextUserID  uuid.UUID
		expectedStatus int
		expectedError  string
		setupMocks     func()
	}{
		{
			name:           "successful delete",
			userID:         userToDelete.ID.String(),
			contextUserID:  adminUser.ID,
			expectedStatus: http.StatusOK,
			setupMocks: func() {
				s.userRepo.EXPECT().GetByID(userToDelete.ID).Return(userToDelete, nil).Times(1)
				s.userRepo.EXPECT().Delete(userToDelete.ID).Return(nil).Times(1)
				s.auditRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:           "cannot delete self",
			userID:         adminUser.ID.String(),
			contextUserID:  adminUser.ID,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot delete your own account",
			setupMocks:     func() {}, // No mocks needed for validation error
		},
		{
			name:           "user not found",
			userID:         uuid.New().String(),
			contextUserID:  adminUser.ID,
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
			setupMocks: func() {
				s.userRepo.EXPECT().GetByID(gomock.Any()).Return(nil, repositories.ErrUserNotFound).Times(1)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup mocks for this test case
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()
			s.userRepo = repository_mocks.NewMockUserRepositoryInterface(ctrl)
			s.auditRepo = repository_mocks.NewMockAuditLogRepositoryInterface(ctrl)
			s.handler = NewAdminHandler(s.userRepo, s.auditRepo)

			tt.setupMocks()

			// Setup request
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%s", tt.userID), nil)
			rec := httptest.NewRecorder()
			c := s.e.NewContext(req, rec)
			c.SetParamNames("userId")
			c.SetParamValues(tt.userID)
			c.Set("user_id", tt.contextUserID)

			// Execute
			err := s.handler.DeleteUser(c)

			// Assert
			if tt.expectedError != "" {
				s.NoError(err) // SendError returns nil, error is in response body
				s.Equal(tt.expectedStatus, rec.Code)

				// Parse and verify error response
				var errorResp ErrorResponse
				parseErr := json.Unmarshal(rec.Body.Bytes(), &errorResp)
				s.NoError(parseErr)

				// Check both message and details for the expected error
				found := strings.Contains(errorResp.Error.Message, tt.expectedError)
				if !found && len(errorResp.Error.Details) > 0 {
					for _, detail := range errorResp.Error.Details {
						if strings.Contains(detail, tt.expectedError) {
							found = true
							break
						}
					}
				}
				s.True(found, "Expected error '%s' not found in message or details", tt.expectedError)
			} else {
				s.NoError(err)
				s.Equal(tt.expectedStatus, rec.Code)
			}
		})
	}
}
