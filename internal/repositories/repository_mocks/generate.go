package repository_mocks

//go:generate mockgen -source=../interfaces.go -destination=repository_mocks.go -package=repository_mocks

// This file contains the go:generate directive to generate mocks for repository interfaces.
// To regenerate the mocks, run:
//   go generate ./internal/repositories/mocks
