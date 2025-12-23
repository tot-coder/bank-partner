package services

import (
	"errors"
	"fmt"
	"log/slog"

	"array-assessment/internal/models"
	"array-assessment/internal/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrNotFound = errors.New("resource not found")
)

type accountSummaryService struct {
	accountRepo repositories.AccountRepositoryInterface
	userRepo    repositories.UserRepositoryInterface
}

func NewAccountSummaryService(
	accountRepo repositories.AccountRepositoryInterface,
	userRepo repositories.UserRepositoryInterface,
) AccountSummaryServiceInterface {
	return &accountSummaryService{
		accountRepo: accountRepo,
		userRepo:    userRepo,
	}
}

func (s *accountSummaryService) GetAccountSummary(requestorID uuid.UUID, targetUserID *uuid.UUID, isAdmin bool) (*models.UserAccountSummary, error) {
	requestor, err := s.validateRequestor(requestorID)
	if err != nil {
		return nil, err
	}

	effectiveUserID, err := s.determineTargetUserID(requestorID, targetUserID, requestor, isAdmin)
	if err != nil {
		return nil, err
	}

	accounts, err := s.fetchUserAccounts(effectiveUserID)
	if err != nil {
		return nil, err
	}

	summary := s.buildAccountSummary(effectiveUserID, accounts)

	slog.Info("account summary generated",
		"user_id", effectiveUserID,
		"account_count", len(accounts),
		"total_balance", summary.TotalBalance.String())

	return summary, nil
}

func (s *accountSummaryService) validateRequestor(requestorID uuid.UUID) (*models.User, error) {
	requestor, err := s.userRepo.GetByID(requestorID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			slog.Warn("user not found during account summary request",
				"requestor_id", requestorID,
				"error", err)
			return nil, ErrNotFound
		}
		slog.Error("failed to get requestor user",
			"requestor_id", requestorID,
			"error", err)
		return nil, fmt.Errorf("failed to verify requestor: %w", err)
	}
	return requestor, nil
}

func (s *accountSummaryService) determineTargetUserID(
	requestorID uuid.UUID,
	targetUserID *uuid.UUID,
	requestor *models.User,
	isAdmin bool,
) (uuid.UUID, error) {
	if targetUserID == nil || *targetUserID == requestorID {
		return requestorID, nil
	}

	if err := s.authorizeAdminAccess(requestorID, *targetUserID, requestor, isAdmin); err != nil {
		return uuid.Nil, err
	}

	if err := s.validateTargetUser(*targetUserID); err != nil {
		return uuid.Nil, err
	}

	s.logAdminAccess(requestorID, requestor.Email, *targetUserID)

	return *targetUserID, nil
}

func (s *accountSummaryService) authorizeAdminAccess(
	requestorID uuid.UUID,
	targetUserID uuid.UUID,
	requestor *models.User,
	isAdmin bool,
) error {
	if !isAdmin || requestor.Role != models.RoleAdmin {
		slog.Warn("unauthorized access attempt to account summary",
			"requestor_id", requestorID,
			"requestor_role", requestor.Role,
			"target_user_id", targetUserID,
			"is_admin", isAdmin)
		return ErrUnauthorized
	}
	return nil
}

func (s *accountSummaryService) validateTargetUser(targetUserID uuid.UUID) error {
	_, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			slog.Warn("target user not found during admin account summary request",
				"target_user_id", targetUserID,
				"error", err)
			return ErrNotFound
		}
		slog.Error("failed to get target user",
			"target_user_id", targetUserID,
			"error", err)
		return fmt.Errorf("failed to verify target user: %w", err)
	}
	return nil
}

func (s *accountSummaryService) logAdminAccess(adminID uuid.UUID, adminEmail string, targetUserID uuid.UUID) {
	slog.Info("admin accessing user account summary",
		"admin_id", adminID,
		"admin_email", adminEmail,
		"target_user_id", targetUserID)
}

func (s *accountSummaryService) fetchUserAccounts(userID uuid.UUID) ([]models.Account, error) {
	accounts, err := s.accountRepo.GetByUserID(userID)
	if err != nil {
		slog.Error("failed to fetch user accounts",
			"user_id", userID,
			"error", err)
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}
	return accounts, nil
}

func (s *accountSummaryService) buildAccountSummary(userID uuid.UUID, accounts []models.Account) *models.UserAccountSummary {
	totalBalance := decimal.Zero
	accountItems := make([]models.AccountSummaryItem, 0, len(accounts))

	for i := range accounts {
		account := &accounts[i]
		totalBalance = totalBalance.Add(account.Balance)
		accountItems = append(accountItems, s.createAccountSummaryItem(account))
	}

	return &models.UserAccountSummary{
		UserID:       userID,
		TotalBalance: totalBalance,
		AccountCount: len(accounts),
		Currency:     "USD",
		Accounts:     accountItems,
		GeneratedAt:  fmt.Sprintf("%d", 0),
	}
}

func (s *accountSummaryService) createAccountSummaryItem(account *models.Account) models.AccountSummaryItem {
	return models.AccountSummaryItem{
		ID:                  account.ID,
		MaskedAccountNumber: maskAccountNumber(account.AccountNumber),
		AccountType:         account.AccountType,
		Balance:             account.Balance,
		Status:              account.Status,
		Currency:            account.Currency,
		InterestRate:        account.InterestRate,
		CreatedAt:           account.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func maskAccountNumber(accountNumber string) string {
	if len(accountNumber) <= 4 {
		return "****"
	}
	lastFour := accountNumber[len(accountNumber)-4:]
	return "****" + lastFour
}
