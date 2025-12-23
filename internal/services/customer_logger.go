package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const (
	// RedactedValue is used to mask sensitive information in logs to avoid logging PII
	RedactedValue = "***REDACTED***"
)

// CustomerLogger provides structured logging for customer-related operations
type CustomerLogger struct {
	logger *slog.Logger
}

// NewCustomerLogger creates a new customer logger
func NewCustomerLogger(logger *slog.Logger) CustomerLoggerInterface {
	return &CustomerLogger{
		logger: logger,
	}
}

// LogCustomerSearchStarted logs the start of a customer search operation
func (cl *CustomerLogger) LogCustomerSearchStarted(ctx context.Context, query string, searchType string, adminUserID uuid.UUID) {
	cl.logger.InfoContext(ctx, "customer search started",
		slog.String("event_type", "customer_search_started"),
		slog.String("query", RedactedValue),  // Mask query to avoid logging PII
		slog.String("search_type", searchType),
		slog.String("admin_user_id", adminUserID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogCustomerSearchCompleted logs the completion of a customer search
func (cl *CustomerLogger) LogCustomerSearchCompleted(ctx context.Context, resultsCount int, durationMs int64) {
	cl.logger.InfoContext(ctx, "customer search completed",
		slog.String("event_type", "customer_search_completed"),
		slog.Int("results_count", resultsCount),
		slog.Int64("duration_ms", durationMs),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogCustomerSearchFailed logs a failed customer search
func (cl *CustomerLogger) LogCustomerSearchFailed(ctx context.Context, errorMsg string, durationMs int64) {
	cl.logger.WarnContext(ctx, "customer search failed",
		slog.String("event_type", "customer_search_failed"),
		slog.String("error", errorMsg),
		slog.Int64("duration_ms", durationMs),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogCustomerCreated logs customer creation
func (cl *CustomerLogger) LogCustomerCreated(ctx context.Context, customerID uuid.UUID, email string, adminUserID uuid.UUID) {
	cl.logger.InfoContext(ctx, "customer created",
		slog.String("event_type", "customer_created"),
		slog.String("customer_id", customerID.String()),
		slog.String("email", email),
		slog.String("admin_user_id", adminUserID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogCustomerProfileUpdated logs customer profile updates
func (cl *CustomerLogger) LogCustomerProfileUpdated(ctx context.Context, customerID uuid.UUID, updatedFields []string, adminUserID uuid.UUID) {
	cl.logger.InfoContext(ctx, "customer profile updated",
		slog.String("event_type", "customer_profile_updated"),
		slog.String("customer_id", customerID.String()),
		slog.Any("updated_fields", updatedFields),
		slog.String("admin_user_id", adminUserID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogCustomerEmailUpdated logs customer email self-service updates
func (cl *CustomerLogger) LogCustomerEmailUpdated(ctx context.Context, customerID uuid.UUID, oldEmail, newEmail string) {
	cl.logger.InfoContext(ctx, "customer email updated",
		slog.String("event_type", "customer_email_updated"),
		slog.String("customer_id", customerID.String()),
		slog.String("old_email", oldEmail),
		slog.String("new_email", newEmail),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogCustomerDeleted logs customer deletion (soft delete)
func (cl *CustomerLogger) LogCustomerDeleted(ctx context.Context, customerID uuid.UUID, accountsDeactivated int, adminUserID uuid.UUID) {
	cl.logger.InfoContext(ctx, "customer deleted",
		slog.String("event_type", "customer_deleted"),
		slog.String("customer_id", customerID.String()),
		slog.Int("accounts_deactivated", accountsDeactivated),
		slog.String("admin_user_id", adminUserID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogPasswordReset logs admin password reset
func (cl *CustomerLogger) LogPasswordReset(ctx context.Context, customerID uuid.UUID, adminUserID uuid.UUID) {
	cl.logger.InfoContext(ctx, "password reset",
		slog.String("event_type", "password_reset"),
		slog.String("customer_id", customerID.String()),
		slog.String("admin_user_id", adminUserID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogPasswordChanged logs self-service password change
func (cl *CustomerLogger) LogPasswordChanged(ctx context.Context, customerID uuid.UUID) {
	cl.logger.InfoContext(ctx, "password changed",
		slog.String("event_type", "password_changed"),
		slog.String("customer_id", customerID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogAccountOwnershipTransferred logs account ownership transfer
func (cl *CustomerLogger) LogAccountOwnershipTransferred(ctx context.Context, accountID, fromCustomerID, toCustomerID, adminUserID uuid.UUID) {
	cl.logger.InfoContext(ctx, "account ownership transferred",
		slog.String("event_type", "account_ownership_transferred"),
		slog.String("account_id", accountID.String()),
		slog.String("from_customer_id", fromCustomerID.String()),
		slog.String("to_customer_id", toCustomerID.String()),
		slog.String("admin_user_id", adminUserID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogAccountCreatedForCustomer logs account creation for a customer
func (cl *CustomerLogger) LogAccountCreatedForCustomer(ctx context.Context, accountID, customerID, adminUserID uuid.UUID, accountNumber string) {
	cl.logger.InfoContext(ctx, "account created for customer",
		slog.String("event_type", "account_created_for_customer"),
		slog.String("account_id", accountID.String()),
		slog.String("customer_id", customerID.String()),
		slog.String("account_number", maskAccountNumber(accountNumber)),
		slog.String("admin_user_id", adminUserID.String()),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogValidationFailure logs validation failures
func (cl *CustomerLogger) LogValidationFailure(ctx context.Context, operation string, errorMsg string) {
	cl.logger.WarnContext(ctx, "validation failure",
		slog.String("event_type", "validation_failure"),
		slog.String("operation", operation),
		slog.String("error", errorMsg),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// LogAuthorizationFailure logs authorization failures
func (cl *CustomerLogger) LogAuthorizationFailure(ctx context.Context, operation string, userID uuid.UUID, requiredRole string) {
	cl.logger.WarnContext(ctx, "authorization failure",
		slog.String("event_type", "authorization_failure"),
		slog.String("operation", operation),
		slog.String("user_id", userID.String()),
		slog.String("required_role", requiredRole),
		slog.Time("timestamp", time.Now()),
		slog.String("request_id", getRequestID(ctx)),
	)
}

// Helper functions

func getRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

