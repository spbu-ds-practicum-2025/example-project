package server

import "github.com/spbu-ds-practicum-2025/example-project/services/api-gateway/internal/models"

// Type aliases to make generated server code work with models package
type (
	AccountIdParam                = models.AccountIdParam
	GetAccountOperationsParams    = models.GetAccountOperationsParams
	TopUpAccountParams            = models.TopUpAccountParams
	TransferBetweenAccountsParams = models.TransferBetweenAccountsParams
	IdempotencyKeyHeader          = models.IdempotencyKeyHeader
)
