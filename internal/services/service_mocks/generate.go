package service_mocks

//go:generate mockgen -source=../interfaces.go -destination=service_mocks.go -package=service_mocks

// This file contains the go:generate directive to generate mocks for service interfaces.
// To regenerate the mocks, run:
//   go generate ./internal/services/mocks
