package services

import (
	"errors"
	"testing"
	"time"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories/repository_mocks"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// AuditServiceTestSuite is the test suite for AuditService
type AuditServiceTestSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	mockRepo *repository_mocks.MockAuditLogRepositoryInterface
	service  AuditServiceInterface
}

func (s *AuditServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = repository_mocks.NewMockAuditLogRepositoryInterface(s.ctrl)
	s.service = NewAuditService(s.mockRepo)
}

func (s *AuditServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestAuditServiceSuite(t *testing.T) {
	suite.Run(t, new(AuditServiceTestSuite))
}

func (s *AuditServiceTestSuite) TestNewAuditService() {
	service := NewAuditService(s.mockRepo)
	s.NotNil(service)
}

func (s *AuditServiceTestSuite) TestValidateActivityType_ValidLogin() {
	err := ValidateActivityType(models.AuditActionLogin)
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestValidateActivityType_ValidProfileUpdated() {
	err := ValidateActivityType(models.AuditActionProfileUpdated)
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestValidateActivityType_ValidCustomerCreated() {
	err := ValidateActivityType(models.AuditActionCustomerCreated)
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestValidateActivityType_InvalidAction() {
	err := ValidateActivityType("invalid_action")
	s.Error(err)
}

func (s *AuditServiceTestSuite) TestValidateActivityType_EmptyAction() {
	err := ValidateActivityType("")
	s.Error(err)
}

func (s *AuditServiceTestSuite) TestCreateAuditLog_ValidLog() {
	userID := uuid.New()
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionLogin,
		Resource:   "auth",
		ResourceID: userID.String(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(l *models.AuditLog) error {
			// Simulate DB behavior: set ID and ensure CreatedAt is set
			l.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.CreateAuditLog(log)
	s.NoError(err)
	s.NotEqual(uuid.Nil, log.ID)
}

func (s *AuditServiceTestSuite) TestCreateAuditLog_NilLog() {
	err := s.service.CreateAuditLog(nil)
	s.Error(err)
	s.ErrorIs(err, ErrInvalidAuditLog)
}

func (s *AuditServiceTestSuite) TestCreateAuditLog_InvalidActivityType() {
	userID := uuid.New()
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     "invalid_action",
		Resource:   "auth",
		ResourceID: userID.String(),
	}

	err := s.service.CreateAuditLog(log)
	s.Error(err)
}

func (s *AuditServiceTestSuite) TestCreateAuditLog_RepositoryError() {
	userID := uuid.New()
	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionLogin,
		Resource:   "auth",
		ResourceID: userID.String(),
	}

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		Return(errors.New("database error")).
		Times(1)

	err := s.service.CreateAuditLog(log)
	s.Error(err)
	s.Contains(err.Error(), "failed to create audit log")
}

func (s *AuditServiceTestSuite) TestGetCustomerActivity_GetAll() {
	userID := uuid.New()
	now := time.Now()

	expectedLogs := []*models.AuditLog{
		{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     models.AuditActionLogout,
			Resource:   "auth",
			ResourceID: userID.String(),
			CreatedAt:  now.Add(-1 * time.Hour),
		},
		{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     models.AuditActionProfileUpdated,
			Resource:   "customer",
			ResourceID: userID.String(),
			CreatedAt:  now.Add(-2 * time.Hour),
		},
		{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     models.AuditActionLogin,
			Resource:   "auth",
			ResourceID: userID.String(),
			CreatedAt:  now.Add(-3 * time.Hour),
		},
	}

	s.mockRepo.EXPECT().
		GetCustomerActivity(userID, nil, nil, 0, 10).
		Return(expectedLogs, int64(3), nil).
		Times(1)

	results, total, err := s.service.GetCustomerActivity(userID, nil, nil, 0, 10)
	s.NoError(err)
	s.Len(results, 3)
	s.Equal(int64(3), total)
	s.Equal(expectedLogs, results)
}

func (s *AuditServiceTestSuite) TestGetCustomerActivity_WithDateRange() {
	userID := uuid.New()
	now := time.Now()
	startDate := now.Add(-2*time.Hour - 30*time.Minute)
	endDate := now.Add(-30 * time.Minute)

	expectedLogs := []*models.AuditLog{
		{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     models.AuditActionLogout,
			Resource:   "auth",
			ResourceID: userID.String(),
			CreatedAt:  now.Add(-1 * time.Hour),
		},
		{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     models.AuditActionProfileUpdated,
			Resource:   "customer",
			ResourceID: userID.String(),
			CreatedAt:  now.Add(-2 * time.Hour),
		},
	}

	s.mockRepo.EXPECT().
		GetCustomerActivity(userID, &startDate, &endDate, 0, 10).
		Return(expectedLogs, int64(2), nil).
		Times(1)

	results, total, err := s.service.GetCustomerActivity(userID, &startDate, &endDate, 0, 10)
	s.NoError(err)
	s.Len(results, 2)
	s.Equal(int64(2), total)
}

func (s *AuditServiceTestSuite) TestGetCustomerActivity_WithPagination() {
	userID := uuid.New()
	now := time.Now()

	expectedLogs := []*models.AuditLog{
		{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     models.AuditActionProfileUpdated,
			Resource:   "customer",
			ResourceID: userID.String(),
			CreatedAt:  now.Add(-2 * time.Hour),
		},
		{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     models.AuditActionLogin,
			Resource:   "auth",
			ResourceID: userID.String(),
			CreatedAt:  now.Add(-3 * time.Hour),
		},
	}

	s.mockRepo.EXPECT().
		GetCustomerActivity(userID, nil, nil, 1, 2).
		Return(expectedLogs, int64(3), nil).
		Times(1)

	results, total, err := s.service.GetCustomerActivity(userID, nil, nil, 1, 2)
	s.NoError(err)
	s.Len(results, 2)
	s.Equal(int64(3), total)
}

func (s *AuditServiceTestSuite) TestGetCustomerActivity_InvalidUserID() {
	results, total, err := s.service.GetCustomerActivity(uuid.Nil, nil, nil, 0, 10)
	s.Error(err)
	s.Len(results, 0)
	s.Equal(int64(0), total)
	s.ErrorIs(err, ErrInvalidUserID)
}

func (s *AuditServiceTestSuite) TestGetCustomerActivity_InvalidDateRange() {
	userID := uuid.New()
	now := time.Now()
	startDate := now
	endDate := now.Add(-1 * time.Hour)

	results, total, err := s.service.GetCustomerActivity(userID, &startDate, &endDate, 0, 10)
	s.Error(err)
	s.Len(results, 0)
	s.Equal(int64(0), total)
	s.ErrorIs(err, ErrAuditDateRange)
}

func (s *AuditServiceTestSuite) TestGetCustomerActivity_RepositoryError() {
	userID := uuid.New()

	s.mockRepo.EXPECT().
		GetCustomerActivity(userID, nil, nil, 0, 10).
		Return(nil, int64(0), errors.New("database error")).
		Times(1)

	results, total, err := s.service.GetCustomerActivity(userID, nil, nil, 0, 10)
	s.Error(err)
	s.Nil(results)
	s.Equal(int64(0), total)
	s.Contains(err.Error(), "database error")
}

func (s *AuditServiceTestSuite) TestLogLogin() {
	userID := uuid.New()

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&userID, log.UserID)
			s.Equal(models.AuditActionLogin, log.Action)
			s.Equal("192.168.1.1", log.IPAddress)
			s.Equal("Mozilla/5.0", log.UserAgent)
			s.Equal("auth", log.Resource)
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogLogin(userID, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestLogLogout() {
	userID := uuid.New()

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&userID, log.UserID)
			s.Equal(models.AuditActionLogout, log.Action)
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogLogout(userID, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestLogProfileUpdate() {
	userID := uuid.New()
	performedBy := uuid.New()

	changes := map[string]interface{}{
		"first_name": "John",
		"last_name":  "Doe",
	}

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&userID, log.UserID)
			s.Equal(models.AuditActionProfileUpdated, log.Action)
			s.NotNil(log.Metadata)
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogProfileUpdate(userID, performedBy, "192.168.1.1", "Mozilla/5.0", changes)
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestLogEmailUpdate() {
	userID := uuid.New()
	performedBy := uuid.New()

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&userID, log.UserID)
			s.Equal(models.AuditActionEmailUpdated, log.Action)
			s.NotNil(log.Metadata)
			s.Equal("old@example.com", log.GetMetadata("old_email", ""))
			s.Equal("new@example.com", log.GetMetadata("new_email", ""))
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogEmailUpdate(userID, performedBy, "old@example.com", "new@example.com", "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestLogPasswordReset() {
	userID := uuid.New()
	performedBy := uuid.New()

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&userID, log.UserID)
			s.Equal(models.AuditActionPasswordReset, log.Action)
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogPasswordReset(userID, performedBy, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestLogPasswordUpdate() {
	userID := uuid.New()

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&userID, log.UserID)
			s.Equal(models.AuditActionPasswordUpdated, log.Action)
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogPasswordUpdate(userID, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestLogCustomerCreated() {
	userID := uuid.New()
	performedBy := uuid.New()

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&userID, log.UserID)
			s.Equal(models.AuditActionCustomerCreated, log.Action)
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogCustomerCreated(userID, performedBy, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestLogCustomerDeleted() {
	userID := uuid.New()
	performedBy := uuid.New()

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&userID, log.UserID)
			s.Equal(models.AuditActionCustomerDeleted, log.Action)
			s.NotNil(log.Metadata)
			s.Equal("Requested by user", log.GetMetadata("reason", ""))
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogCustomerDeleted(userID, performedBy, "192.168.1.1", "Mozilla/5.0", "Requested by user")
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestLogAccountCreated() {
	userID := uuid.New()
	performedBy := uuid.New()
	accountID := uuid.New()

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&userID, log.UserID)
			s.Equal(models.AuditActionAccountCreated, log.Action)
			s.Equal("account", log.Resource)
			s.Equal(accountID.String(), log.ResourceID)
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogAccountCreated(userID, performedBy, accountID, "checking", "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}

func (s *AuditServiceTestSuite) TestLogAccountTransferred() {
	fromUserID := uuid.New()
	toUserID := uuid.New()
	performedBy := uuid.New()
	accountID := uuid.New()

	s.mockRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(log *models.AuditLog) error {
			s.Equal(&fromUserID, log.UserID)
			s.Equal(models.AuditActionAccountTransferred, log.Action)
			s.Equal(fromUserID.String(), log.GetMetadata("from_user_id", ""))
			s.Equal(toUserID.String(), log.GetMetadata("to_user_id", ""))
			log.ID = uuid.New()
			return nil
		}).
		Times(1)

	err := s.service.LogAccountTransferred(fromUserID, toUserID, performedBy, accountID, "192.168.1.1", "Mozilla/5.0")
	s.NoError(err)
}
