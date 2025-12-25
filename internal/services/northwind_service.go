package services

import (
	"array-assessment/internal/config"
	"array-assessment/internal/dto"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type AuthTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())

	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")

	return t.base.RoundTrip(req)
}

// NorthWindService handles customer search operations
type NorthWindService struct {
	config *config.NorthWindConfig
	client *http.Client
	logger *slog.Logger
}

// NewNorthWindService creates a new NorthWind service
func NewNorthWindService(
	cfg *config.NorthWindConfig,
	logger *slog.Logger,
) NorthWindServiceInterface {

	transport := &AuthTransport{
		apiKey: cfg.ApiKey,
		base:   http.DefaultTransport,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return &NorthWindService{
		config: cfg,
		client: client,
		logger: logger,
	}
}

func (s *NorthWindService) buildRequest(
	ctx context.Context,
	method, path string,
	body any,
) (*http.Request, error) {

	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		buf = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		s.config.BaseUrl+path,
		buf,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	return req, nil
}

func (s *NorthWindService) do(req *http.Request) (*http.Response, []byte, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error(
			"northwind request failed",
			"method", req.Method,
			"url", req.URL.String(),
			"error", err,
		)
		return nil, nil, err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, nil, fmt.Errorf("read response body: %w", err)
	}

	return resp, body, nil
}

func (s *NorthWindService) AuthAccount(ctx context.Context, requestDto dto.NorthWindAccountRequestDto) (*dto.NorthWindAccountValidationResult, error) {
	req, err := s.buildRequest(
		ctx,
		http.MethodPost,
		"/external/accounts/validate",
		requestDto,
	)

	s.logger.Info("AAAAAAAAAAAAAAA", requestDto.AccountHolderName)

	if err != nil {
		return nil, err
	}

	resp, body, err := s.do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {

	case http.StatusOK:
		var success dto.NorthWindValidateResponse[dto.NorthWindAccountData]
		if err := json.Unmarshal(body, &success); err != nil {
			return nil, fmt.Errorf("decode success response: %w", err)
		}

		accountExists := success.Data != nil && success.Data.AccountID != ""
		accountValid := success.Validation.Valid

		s.logger.Info(
			"northwind validation result",
			"account_exists", accountExists,
			"account_valid", accountValid,
			"account_id", success.Data.AccountID,
		)

		return &dto.NorthWindAccountValidationResult{
			Response:         &success,
			AvailableBalance: success.Data.AvailableBalance,
			AccountExists:    accountExists,
			AccountValid:     accountValid,
		}, nil

	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusInternalServerError:

		var errResp dto.NorthwindValidateAccountErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf(
				"northwind error (%d): %s",
				resp.StatusCode,
				string(body),
			)
		}

		s.logger.Error(
			"northwind validation error",
			"status", resp.StatusCode,
			"code", errResp.Error.Code,
			"message", errResp.Error.Message,
			"request_id", errResp.Error.RequestID,
		)

		return nil, errors.New(errResp.Error.Message)

	default:
		return nil, fmt.Errorf(
			"unexpected northwind response (%d): %s",
			resp.StatusCode,
			string(body),
		)
	}
}
