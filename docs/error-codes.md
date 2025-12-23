# API Error Codes Reference

This document catalogs all error codes used in the Banking API, organized by category. Each error includes its code, HTTP status, message, and usage context.

## Table of Contents

- [Error Response Format](#error-response-format)
- [HTTP Status Code Reference](#http-status-code-reference)
- [Authentication Errors (AUTH_*)](#authentication-errors-auth_)
- [Validation Errors (VALIDATION_*)](#validation-errors-validation_)
- [Customer Errors (CUSTOMER_*)](#customer-errors-customer_)
- [Account Errors (ACCOUNT_*)](#account-errors-account_)
- [Transaction Errors (TRANSACTION_*)](#transaction-errors-transaction_)
- [System Errors (SYSTEM_*)](#system-errors-system_)
- [Example Responses](#example-responses)

## Error Response Format

All error responses follow this standardized JSON structure:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": ["Optional array of additional information"],
    "trace_id": "uuid-for-request-tracking"
  }
}
```

### Response Headers

All error responses include:
- `Content-Type: application/json`
- `X-Trace-ID: <uuid>` (matches trace_id in response body)

## HTTP Status Code Reference

| Status Code | Category | When to Use |
|------------|----------|-------------|
| **400** | Bad Request | Malformed JSON, invalid data types, general validation failures |
| **401** | Unauthorized | Missing/invalid/expired authentication token |
| **403** | Forbidden | Valid authentication but insufficient permissions |
| **404** | Not Found | Requested resource does not exist |
| **422** | Unprocessable Entity | Valid request format but semantic/business logic validation failure |
| **429** | Too Many Requests | Rate limit exceeded |
| **500** | Internal Server Error | Unexpected system errors, unhandled exceptions |
| **503** | Service Unavailable | System maintenance or temporary service unavailability |

---

## Authentication Errors (AUTH_*)

### AUTH_001: Invalid Credentials
- **HTTP Status**: 401 Unauthorized
- **Message**: "Invalid email or password"
- **When Used**: Login attempt with incorrect credentials
- **Endpoints**: `POST /api/v1/auth/login`

### AUTH_002: Missing Authorization Token
- **HTTP Status**: 401 Unauthorized
- **Message**: "Authorization token is required"
- **When Used**: Request to protected endpoint without Bearer token
- **Endpoints**: All protected endpoints

### AUTH_003: Expired Authorization Token
- **HTTP Status**: 401 Unauthorized
- **Message**: "Authorization token has expired"
- **When Used**: Request with expired JWT token
- **Endpoints**: All protected endpoints

### AUTH_004: Invalid Authorization Token Format
- **HTTP Status**: 401 Unauthorized
- **Message**: "Invalid authorization token format"
- **When Used**: Malformed Bearer token, invalid JWT structure
- **Endpoints**: All protected endpoints

### AUTH_005: Insufficient Permissions
- **HTTP Status**: 403 Forbidden
- **Message**: "Insufficient permissions to access this resource"
- **When Used**: User authenticated but lacks required role/permissions
- **Endpoints**: Admin endpoints, resource access validation

### AUTH_006: Account Locked
- **HTTP Status**: 403 Forbidden
- **Message**: "Account is locked or disabled"
- **When Used**: Account locked due to failed login attempts or administrative action
- **Endpoints**: `POST /api/v1/auth/login`

---

## Validation Errors (VALIDATION_*)

### VALIDATION_001: General Validation Failure
- **HTTP Status**: 400 Bad Request
- **Message**: "Validation failed"
- **When Used**: Multiple field validation errors, malformed request body
- **Details**: Array of field-specific error messages
- **Endpoints**: All endpoints accepting request bodies

### VALIDATION_002: Required Field Missing
- **HTTP Status**: 400 Bad Request
- **Message**: "Required field is missing"
- **When Used**: Specific required field not provided
- **Endpoints**: All endpoints with required fields

### VALIDATION_003: Invalid Field Format
- **HTTP Status**: 400 Bad Request
- **Message**: "Invalid field format"
- **When Used**: Field value doesn't match expected format
- **Endpoints**: All endpoints with formatted fields

### VALIDATION_004: Field Value Out of Range
- **HTTP Status**: 400 Bad Request
- **Message**: "Field value is out of allowed range"
- **When Used**: Numeric or date values outside acceptable range
- **Endpoints**: Pagination, amount fields

### VALIDATION_005: Invalid Email Format
- **HTTP Status**: 400 Bad Request
- **Message**: "Invalid email address format"
- **When Used**: Email field doesn't match email pattern
- **Endpoints**: Registration, profile updates

### VALIDATION_006: Invalid Phone Format
- **HTTP Status**: 400 Bad Request
- **Message**: "Invalid phone number format"
- **When Used**: Phone number doesn't match expected format
- **Endpoints**: Profile updates

### VALIDATION_007: Invalid Date Format or Range
- **HTTP Status**: 400 Bad Request
- **Message**: "Invalid date format or range"
- **When Used**: Date parameter malformed or outside valid range
- **Endpoints**: Transaction filters, search endpoints

---

## Customer Errors (CUSTOMER_*)

### CUSTOMER_001: Customer Not Found
- **HTTP Status**: 404 Not Found
- **Message**: "Customer not found"
- **When Used**: Customer ID doesn't exist in system
- **Endpoints**: `GET /api/v1/customers/:id`, `PUT /api/v1/customers/:id`

### CUSTOMER_002: Customer Already Exists
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "An account with this email already exists"
- **When Used**: Registration or update with duplicate email
- **Endpoints**: `POST /api/v1/auth/register`, `PUT /api/v1/customers/:id`

### CUSTOMER_003: Customer Account Inactive
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "Customer account is inactive or suspended"
- **When Used**: Operation attempted on inactive/suspended account
- **Endpoints**: Transaction operations, account access

### CUSTOMER_004: Invalid Customer ID Format
- **HTTP Status**: 400 Bad Request
- **Message**: "Invalid customer ID format"
- **Details**: ["ID must be a valid UUID"]
- **When Used**: Customer ID parameter is not a valid UUID
- **Endpoints**: All endpoints with :id parameter

### CUSTOMER_005: Customer Search No Results
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "Customer search returned no results"
- **When Used**: Search query matched no customers
- **Endpoints**: `POST /api/v1/customers/search`

---

## Account Errors (ACCOUNT_*)

### ACCOUNT_001: Account Not Found
- **HTTP Status**: 404 Not Found
- **Message**: "Account not found"
- **When Used**: Account ID doesn't exist in system
- **Endpoints**: Account operations, transaction operations

### ACCOUNT_002: Account Inactive
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "Account is closed or inactive"
- **When Used**: Operation attempted on closed/frozen account
- **Endpoints**: Transaction creation, transfers

### ACCOUNT_003: Insufficient Account Balance
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "Insufficient account balance"
- **When Used**: Debit/withdrawal exceeds available balance
- **Endpoints**: Transaction creation, transfers

### ACCOUNT_004: Invalid Account Number
- **HTTP Status**: 400 Bad Request
- **Message**: "Invalid account number or type"
- **When Used**: Account identifier malformed or invalid
- **Endpoints**: All endpoints with account ID parameters

### ACCOUNT_005: Account Operation Not Permitted
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "Account operation not permitted"
- **When Used**: Operation violates account type rules or restrictions
- **Endpoints**: Account management, transaction operations

---

## Transaction Errors (TRANSACTION_*)

### TRANSACTION_001: Transaction Not Found
- **HTTP Status**: 404 Not Found
- **Message**: "Transaction not found"
- **When Used**: Transaction ID doesn't exist in system
- **Endpoints**: `GET /api/v1/accounts/:accountId/transactions/:id`

### TRANSACTION_002: Invalid Transaction Amount
- **HTTP Status**: 400 Bad Request
- **Message**: "Invalid transaction amount"
- **Details**: ["Amount must be greater than 0"]
- **When Used**: Amount is negative, zero, or improperly formatted
- **Endpoints**: `POST /api/v1/accounts/:id/transactions`

### TRANSACTION_003: Insufficient Funds
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "Insufficient account balance for this transaction"
- **Details**: ["Required: $X.XX, Available: $Y.YY"]
- **When Used**: Transaction amount exceeds account balance
- **Endpoints**: `POST /api/v1/accounts/:id/transactions`

### TRANSACTION_004: Duplicate Transaction
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "Transaction with this idempotency key already exists"
- **When Used**: Idempotency key matches existing transaction
- **Endpoints**: `POST /api/v1/transactions`

### TRANSACTION_005: Transaction Validation Failed
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "Transaction validation failed"
- **When Used**: Business rule validation failure
- **Endpoints**: Transaction operations

### TRANSACTION_006: Invalid Transaction Type
- **HTTP Status**: 422 Unprocessable Entity
- **Message**: "Invalid transaction type"
- **When Used**: Transaction type not recognized or not allowed
- **Endpoints**: `POST /api/v1/accounts/:id/transactions`

---

## System Errors (SYSTEM_*)

### SYSTEM_001: Internal Server Error
- **HTTP Status**: 500 Internal Server Error
- **Message**: "An unexpected error occurred. Please contact support with trace ID"
- **When Used**: Unhandled exceptions, unexpected errors
- **Details**: Never includes internal error details
- **Endpoints**: All endpoints (catch-all)

### SYSTEM_002: Database Connection Error
- **HTTP Status**: 500 Internal Server Error
- **Message**: "Database connection error"
- **When Used**: Database connectivity issues
- **Endpoints**: All endpoints requiring database access

### SYSTEM_003: External Service Unavailable
- **HTTP Status**: 503 Service Unavailable
- **Message**: "Service temporarily unavailable"
- **Details**: ["Database connection failed"]
- **When Used**: Dependent service down or unreachable
- **Endpoints**: `GET /health`, all endpoints

### SYSTEM_004: Configuration Error
- **HTTP Status**: 500 Internal Server Error
- **Message**: "System configuration error"
- **When Used**: Missing or invalid system configuration
- **Endpoints**: System initialization, startup

### SYSTEM_005: Unexpected Error
- **HTTP Status**: 500 Internal Server Error
- **Message**: "An unexpected error occurred"
- **When Used**: Errors that don't fit other categories
- **Endpoints**: All endpoints (catch-all)

### SYSTEM_006: Rate Limit Exceeded
- **HTTP Status**: 429 Too Many Requests
- **Message**: "Rate limit exceeded. Please try again later"
- **Details**: ["Limit: 5 requests per second"]
- **When Used**: Request rate exceeds configured limits (5 req/sec per IP)
- **Endpoints**: All endpoints (enforced by middleware)

---

## Example Responses

### Authentication Error Example

```json
POST /api/v1/auth/login
Status: 401 Unauthorized

{
  "error": {
    "code": "AUTH_001",
    "message": "Invalid email or password",
    "details": [],
    "trace_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

### Validation Error Example

```json
POST /api/v1/auth/register
Status: 400 Bad Request

{
  "error": {
    "code": "VALIDATION_001",
    "message": "Validation failed",
    "details": [
      "email: must be a valid email address",
      "password: must be at least 8 characters long",
      "first_name: is required"
    ],
    "trace_id": "550e8400-e29b-41d4-a716-446655440001"
  }
}
```

### Transaction Error Example

```json
POST /api/v1/accounts/123e4567-e89b-12d3-a456-426614174000/transactions
Status: 422 Unprocessable Entity

{
  "error": {
    "code": "TRANSACTION_003",
    "message": "Insufficient account balance for this transaction",
    "details": ["Required: $500.00, Available: $250.00"],
    "trace_id": "550e8400-e29b-41d4-a716-446655440002"
  }
}
```

### System Error Example

```json
GET /health
Status: 503 Service Unavailable

{
  "error": {
    "code": "SYSTEM_003",
    "message": "Service temporarily unavailable",
    "details": ["Database connection failed"],
    "trace_id": "550e8400-e29b-41d4-a716-446655440003"
  }
}
```

---

## Error Handling Best Practices

### For API Consumers

1. **Always check HTTP status code** first to understand the error category
2. **Parse the error code** for programmatic handling of specific scenarios
3. **Display the message** to end users - it's human-readable and safe
4. **Log the trace_id** for support requests and debugging
5. **Use details array** for field-specific validation feedback

### For API Developers

1. **Never expose internal errors** - use WrapSystemError() for database/system errors
2. **Include trace IDs** in all error responses for request tracking
3. **Provide actionable details** - tell users what they need to fix
4. **Log full error context** server-side with trace ID for debugging
5. **Use appropriate HTTP status codes** matching the error category

## Security Considerations

- **No sensitive data** in error messages (no database schema, stack traces, credentials)
- **Generic system errors** for 500-level errors to prevent information disclosure
- **Consistent timing** for authentication errors to prevent user enumeration
- **Rate limiting** included in error codes to prevent abuse
- **Audit logging** of all errors server-side with full context

---

Generated: 2025-10-21
Version: 1.0
Last Updated: 2025-10-21
