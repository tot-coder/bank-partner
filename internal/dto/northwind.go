package dto

import (
	"time"

	"github.com/shopspring/decimal"
)

type NorthWindAccountRequestDto struct {
	AccountHolderName string `json:"account_holder_name"`
	AccountNumber     string `json:"account_number"`
	RoutingNumber     string `json:"routing_number"`
}

// ---------- Top-Level Response ----------

type NorthWindValidateResponse[T any] struct {
	Validation NorthWindValidation `json:"validation"`
	Data       *T                  `json:"data,omitempty"`
}

// ---------- Validation ----------

type NorthWindValidation struct {
	Valid          bool                            `json:"valid"`
	Issues         []NorthWindValidateAccountIssue `json:"issues,omitempty"`
	Metadata       map[string]any                  `json:"metadata,omitempty"`
	ValidationTime time.Time                       `json:"validation_time"`
}

type NorthWindValidateAccountIssue struct {
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// ---------- Account Data ----------

type NorthWindAccountData struct {
	AccountHolderName string          `json:"account_holder_name"`
	AccountID         string          `json:"account_id"`
	AccountStatus     string          `json:"account_status"`
	AccountType       string          `json:"account_type"`
	AvailableBalance  decimal.Decimal `json:"available_balance"`
}

type NorthwindValidateAccountErrorResponse struct {
	Error NorthwindValidateAccountErrorDetail `json:"error"`
}

type NorthwindValidateAccountErrorDetail struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details"`
	RequestID string         `json:"request_id"`
	Timestamp string         `json:"timestamp"`
}

type NorthWindAccountValidationResult struct {
	Response         *NorthWindValidateResponse[NorthWindAccountData] `json:"response"`
	AvailableBalance decimal.Decimal
	AccountExists    bool
	AccountValid     bool
}

// ---------- Transfer ----------

type NorthWindTransferValidationRequest struct {
	Amount             float64                  `json:"amount"`
	Currency           string                   `json:"currency"`
	DestinationAccount NorthWindTransferAccount `json:"destination_account"`
	SourceAccount      NorthWindTransferAccount `json:"source_account"`
	Description        string                   `json:"description"`
	Direction          string                   `json:"direction"`
	ReferenceNumber    string                   `json:"reference_number"`
	TransferType       string                   `json:"transfer_type"`
	ScheduledDate      string                   `json:"scheduled_date"`
}

type NorthWindTransferAccount struct {
	AccountHolderName string  `json:"account_holder_name"`
	AccountNumber     string  `json:"account_number"`
	RoutingNumber     string  `json:"routing_number"`
	InstitutionName   *string `json:"institution_name,omitempty"`
}

type NorthWindTransferResponseData struct {
	Amount                 decimal.Decimal `json:"amount"`
	Currency               string          `json:"currency"`
	Direction              string          `json:"direction"`
	EstimateFee            decimal.Decimal `json:"estimate_fee"`
	EstimateTime           string          `json:"estimate_time"`
	ExpectedCompletionDate time.Time       `json:"expected_completion_date"`
	ReferenceNumber        string          `json:"reference_number"`
	TransferType           string          `json:"transfer_type"`
}

type NorthWindInitiateTransferRequest struct {
	Amount             float64                  `json:"amount"`
	Currency           string                   `json:"currency"`
	Direction          string                   `json:"direction"`
	DestinationAccount NorthWindTransferAccount `json:"destination_account"`
	SourceAccount      NorthWindTransferAccount `json:"source_account"`
	Description        string                   `json:"description"`
	ReferenceNumber    string                   `json:"reference_number"`
	TransferType       string                   `json:"transfer_type"`
	ScheduledDate      string                   `json:"scheduled_date"`
}

// ---------- Status History ----------

type NorthWindTransferStatusHistory struct {
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

// ---------- Initiate Transfer Response ----------

type NorthWindTransferStatusResponse struct {
	TransferID             string                           `json:"transfer_id"`
	Status                 string                           `json:"status"`
	ReferenceNumber        string                           `json:"reference_number"`
	Amount                 decimal.Decimal                  `json:"amount"` // use decimal for precision
	Currency               string                           `json:"currency"`
	Direction              string                           `json:"direction"`
	TransferType           string                           `json:"transfer_type"`
	Description            string                           `json:"description"`
	InitiatedDate          time.Time                        `json:"initiated_date"`
	ExpectedCompletionDate time.Time                        `json:"expected_completion_date"`
	Fee                    decimal.Decimal                  `json:"fee"`           // use decimal for money
	ExchangeRate           decimal.Decimal                  `json:"exchange_rate"` // use decimal for precision
	RetryCount             int                              `json:"retry_count"`
	SourceAccount          NorthWindTransferAccount         `json:"source_account"`
	DestinationAccount     NorthWindTransferAccount         `json:"destination_account"`
	StatusHistory          []NorthWindTransferStatusHistory `json:"status_history"`
}
