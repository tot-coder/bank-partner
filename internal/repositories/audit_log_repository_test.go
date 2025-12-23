package repositories

import (
	"testing"
	"time"

	"array-assessment/internal/database"
	"array-assessment/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

func TestAuditLogRepository(t *testing.T) {
	suite.Run(t, new(AuditLogRepositorySuite))
}

type AuditLogRepositorySuite struct {
	suite.Suite
	db   *database.DB
	repo AuditLogRepositoryInterface
}

func (s *AuditLogRepositorySuite) SetupTest() {
	s.db = database.SetupTestDB(s.T())
	s.repo = NewAuditLogRepository(s.db.DB)
}

func (s *AuditLogRepositorySuite) TearDownTest() {
	database.CleanupTestDB(s.T(), s.db)
}

func (s *AuditLogRepositorySuite) TestAuditLogRepository_Create() {
	userID := uuid.New()

	log := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionLogin,
		Resource:   "user",
		ResourceID: userID.String(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	err := s.repo.Create(log)
	s.NoError(err)
	s.NotEqual(uuid.Nil, log.ID)
	s.NotZero(log.CreatedAt)
}

func (s *AuditLogRepositorySuite) TestAuditLogRepository_CreateWithoutUserID() {
	log := &models.AuditLog{
		UserID:     nil, // Anonymous action
		Action:     models.AuditActionFailedLogin,
		Resource:   "auth",
		ResourceID: "",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}

	err := s.repo.Create(log)
	s.NoError(err)
	s.NotEqual(uuid.Nil, log.ID)
	s.Nil(log.UserID)
}

func (s *AuditLogRepositorySuite) TestAuditLogRepository_GetByUserID() {
	userID := uuid.New()

	// Create multiple logs for the same user
	actions := []string{models.AuditActionLogin, models.AuditActionUpdate, models.AuditActionLogout}
	for _, action := range actions {
		log := &models.AuditLog{
			UserID:     &userID,
			Action:     action,
			Resource:   "user",
			ResourceID: userID.String(),
			IPAddress:  "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
		}
		err := s.repo.Create(log)
		s.NoError(err)
	}

	// Create log for different user
	otherUserID := uuid.New()
	otherLog := &models.AuditLog{
		UserID:     &otherUserID,
		Action:     models.AuditActionLogin,
		Resource:   "user",
		ResourceID: otherUserID.String(),
		IPAddress:  "192.168.1.2",
		UserAgent:  "Chrome",
	}
	err := s.repo.Create(otherLog)
	s.NoError(err)

	// Get logs for first user
	logs, total, err := s.repo.GetByUserID(userID, 0, 10)
	s.NoError(err)
	s.Len(logs, 3)
	s.Equal(int64(3), total)

	// Verify all logs belong to the correct user
	for _, log := range logs {
		s.NotNil(log.UserID)
		s.Equal(userID, *log.UserID)
	}
}

func (s *AuditLogRepositorySuite) TestAuditLogRepository_GetByUserID_Pagination() {
	userID := uuid.New()

	// Create 5 logs
	for i := 0; i < 5; i++ {
		log := &models.AuditLog{
			UserID:     &userID,
			Action:     models.AuditActionUpdate,
			Resource:   "account",
			ResourceID: uuid.New().String(),
			IPAddress:  "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
		}
		err := s.repo.Create(log)
		s.NoError(err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get first page
	logs, total, err := s.repo.GetByUserID(userID, 0, 2)
	s.NoError(err)
	s.Len(logs, 2)
	s.Equal(int64(5), total)

	// Get second page
	logs, total, err = s.repo.GetByUserID(userID, 2, 2)
	s.NoError(err)
	s.Len(logs, 2)
	s.Equal(int64(5), total)

	// Get third page (partial)
	logs, total, err = s.repo.GetByUserID(userID, 4, 2)
	s.NoError(err)
	s.Len(logs, 1)
	s.Equal(int64(5), total)
}

func (s *AuditLogRepositorySuite) TestAuditLogRepository_GetByAction() {
	// Create logs with different actions
	userID1 := uuid.New()
	userID2 := uuid.New()

	loginLog1 := &models.AuditLog{
		UserID:     &userID1,
		Action:     models.AuditActionLogin,
		Resource:   "user",
		ResourceID: userID1.String(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}
	err := s.repo.Create(loginLog1)
	s.NoError(err)

	loginLog2 := &models.AuditLog{
		UserID:     &userID2,
		Action:     models.AuditActionLogin,
		Resource:   "user",
		ResourceID: userID2.String(),
		IPAddress:  "192.168.1.2",
		UserAgent:  "Chrome",
	}
	err = s.repo.Create(loginLog2)
	s.NoError(err)

	updateLog := &models.AuditLog{
		UserID:     &userID1,
		Action:     models.AuditActionUpdate,
		Resource:   "account",
		ResourceID: uuid.New().String(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}
	err = s.repo.Create(updateLog)
	s.NoError(err)

	// Get login actions
	logs, total, err := s.repo.GetByAction(models.AuditActionLogin, 0, 10)
	s.NoError(err)
	s.Len(logs, 2)
	s.Equal(int64(2), total)

	for _, log := range logs {
		s.Equal(models.AuditActionLogin, log.Action)
	}

	// Get update actions
	logs, total, err = s.repo.GetByAction(models.AuditActionUpdate, 0, 10)
	s.NoError(err)
	s.Len(logs, 1)
	s.Equal(int64(1), total)
	s.Equal(models.AuditActionUpdate, logs[0].Action)
}

func (s *AuditLogRepositorySuite) TestAuditLogRepository_GetByResource() {
	userID := uuid.New()
	resourceID := uuid.New().String()

	// Create multiple logs for the same resource
	actions := []string{models.AuditActionCreate, models.AuditActionUpdate, models.AuditActionDelete}
	for _, action := range actions {
		log := &models.AuditLog{
			UserID:     &userID,
			Action:     action,
			Resource:   "account",
			ResourceID: resourceID,
			IPAddress:  "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
		}
		err := s.repo.Create(log)
		s.NoError(err)
	}

	// Create log for different resource
	otherLog := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionUpdate,
		Resource:   "account",
		ResourceID: uuid.New().String(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}
	err := s.repo.Create(otherLog)
	s.NoError(err)

	// Get logs for specific resource
	logs, total, err := s.repo.GetByResource("account", resourceID, 0, 10)
	s.NoError(err)
	s.Len(logs, 3)
	s.Equal(int64(3), total)

	for _, log := range logs {
		s.Equal(resourceID, log.ResourceID)
	}
}

func (s *AuditLogRepositorySuite) TestAuditLogRepository_GetByTimeRange() {
	userID := uuid.New()

	// Create logs at different times
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	// Create yesterday's log
	log1 := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionLogin,
		Resource:   "user",
		ResourceID: userID.String(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}
	err := s.repo.Create(log1)
	s.NoError(err)

	// Create today's log
	log2 := &models.AuditLog{
		UserID:     &userID,
		Action:     models.AuditActionUpdate,
		Resource:   "account",
		ResourceID: uuid.New().String(),
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
	}
	err = s.repo.Create(log2)
	s.NoError(err)

	// Get logs from yesterday to tomorrow
	logs, total, err := s.repo.GetByTimeRange(yesterday, tomorrow, 0, 10)
	s.NoError(err)
	s.GreaterOrEqual(len(logs), 2)
	s.GreaterOrEqual(total, int64(2))

	// Get logs from tomorrow (should be empty)
	logs, total, err = s.repo.GetByTimeRange(tomorrow, tomorrow.Add(24*time.Hour), 0, 10)
	s.NoError(err)
	s.Len(logs, 0)
	s.Equal(int64(0), total)
}
