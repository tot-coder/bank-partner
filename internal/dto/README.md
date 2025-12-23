# DTO Package

This package contains all Data Transfer Objects (DTOs) used in the Banking API.

## Purpose

DTOs serve as the contract between the API and its clients, defining the structure of requests and responses. They provide:
- Clear API documentation through strongly-typed structures
- Request validation via struct tags
- Separation between internal models and external API representations
- Centralized location for all API contracts

## Structure

The DTOs are organized by domain:
- `account.go` - Account management DTOs (create, update, status, summary, transactions, transfers)
- `auth.go` - Authentication DTOs (registration, login, token refresh, user profile)
- `admin.go` - Admin operation DTOs (user management, user unlocking, audit logs)
- `customer.go` - Customer management DTOs (search, profile, create, update, delete)
- `transaction.go` - Transaction DTOs (filtering, pagination, transaction history with balances)
- `queue.go` - Queue metrics DTOs (processing queue statistics)

## Usage

### In Handlers
```go
import "array-assessment/internal/dto"

func (h *Handler) CreateAccount(c echo.Context) error {
    var req dto.CreateAccountRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
    }
    // ...
}
```

### In Tests
```go
reqBody := dto.CreateAccountRequest{
    AccountType:    "checking",
    InitialDeposit: "100.00",
}
```

## Conventions

1. **Naming**: 
   - Request DTOs end with `Request`
   - Response DTOs end with `Response`

2. **Validation**: 
   - Use struct tags for validation rules
   - Required fields use `validate:"required"`
   - Field constraints use appropriate validators

3. **JSON Tags**:
   - All fields must have `json` tags
   - Use camelCase for JSON field names
   - Optional fields use `omitempty`

4. **Documentation**:
   - Each DTO should have a comment describing its purpose
   - Complex fields should have inline documentation

## DTO Reference

### Account DTOs (`account.go`)

**Request DTOs:**
- `CreateAccountRequest` - Create a new account (accountType, initialDeposit)
- `UpdateAccountStatusRequest` - Update account status (status)
- `TransactionRequest` - Perform a transaction (amount, type, description)
- `TransferRequest` - Transfer funds between accounts (toAccountId, amount, description)

**Response DTOs:**
- `CreateAccountResponse` - Account creation result
- `AccountResponse` - Single account details
- `AccountListResponse` - Paginated list of accounts
- `TransactionListResponse` - Paginated list of transactions
- `AccountSummaryResponse` - Aggregated account information with balances by type
- `TransferResponse` - Transfer confirmation with transaction IDs
- `TransferHistoryResponse` - Paginated transfer history
- `MessageResponse` - Simple message response
- `PaginationMeta` - Pagination metadata

### Authentication DTOs (`auth.go`)

**Request DTOs:**
- `RegisterRequest` - User registration (email, password, firstName, lastName)
- `LoginRequest` - Login credentials (email, password)
- `RefreshTokenRequest` - Token refresh (refreshToken)

**Response DTOs:**
- `TokenResponse` - Authentication tokens (accessToken, refreshToken, tokenType, expiresAt)
- `UserProfileResponse` - User profile information

### Admin DTOs (`admin.go`)

**Request DTOs:**
- `UnlockUserRequest` - Unlock a locked user account (userId)
- `ListUsersRequest` - Query parameters for user listing (offset, limit)

**Response DTOs:**
- `UserResponse` - User details including lock status and failed login attempts
- `UsersListResponse` - Paginated list of users
- `AuditLogResponse` - Audit log entry details
- `AuditLogsListResponse` - Paginated list of audit logs

### Customer DTOs (`customer.go`)

**Request DTOs:**
- `SearchCustomersRequest` - Search for customers (query, limit, offset)
- `CreateCustomerRequest` - Create new customer (email, name, phone, address, SSN, employment, income)
- `UpdateCustomerProfileRequest` - Update customer profile (firstName, lastName, phone, address, city, state, zipCode)
- `UpdateCustomerEmailRequest` - Update customer email (newEmail)

**Response DTOs:**
- `SearchCustomersResponse` - Customer search results with pagination
- `CustomerSearchResult` - Individual customer in search results
- `GetCustomerProfileResponse` - Detailed customer profile
- `CreateCustomerResponse` - Customer creation result with temporary password
- `UpdateCustomerEmailResponse` - Email update confirmation
- `DeleteCustomerResponse` - Customer deletion confirmation

### Transaction DTOs (`transaction.go`)

**Request DTOs:**
- `TransactionFilters` - Filter options for transactions (startDate, endDate, type, status, category)
- `PaginationParams` - Cursor-based pagination parameters

**Response DTOs:**
- `TransactionWithBalance` - Transaction details with running balance
- `PaginationInfo` - Cursor pagination metadata (hasMore, nextCursor)
- `ListTransactionsResponse` - Paginated transaction list with balances

### Queue DTOs (`queue.go`)

**Response DTOs:**
- `QueueMetrics` - Processing queue statistics (pending, processing, completed, failed counts, avg processing time)