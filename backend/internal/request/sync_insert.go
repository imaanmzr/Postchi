package request

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
)

// SyncInsertParams holds fields for inserting an OpenAPI-synced request row.
type SyncInsertParams struct {
	CollectionID uuid.UUID
	SpecID       uuid.UUID
	UserID       uuid.UUID
	Name         string
	Method       string
	URL          string
	Headers      []KVPair
	Params       []KVPair
	PathVars     []KVPair
	Body         BodySpec
	Auth         AuthSpec
	Settings     Settings
	SortOrder    int
	Description  string
	OperationID  string
	OpHash       string
	ApiDoc       json.RawMessage
}

// InsertSyncedRequest inserts a request row linked to an OpenAPI spec operation.
func InsertSyncedRequest(ctx context.Context, q *sqlc.Queries, p SyncInsertParams) error {
	headers, _ := json.Marshal(p.Headers)
	params, _ := json.Marshal(p.Params)
	pathVars, _ := json.Marshal(p.PathVars)
	body, _ := json.Marshal(p.Body)
	authB, _ := json.Marshal(p.Auth)
	settings, _ := json.Marshal(p.Settings)
	apiDoc := p.ApiDoc
	if len(apiDoc) == 0 {
		apiDoc = []byte("{}")
	}
	return q.InsertSyncedRequest(ctx, sqlc.InsertSyncedRequestParams{
		CollectionID:      pgtype.UUID{Bytes: p.CollectionID, Valid: true},
		Name:              p.Name,
		Method:            p.Method,
		Url:               p.URL,
		Headers:           headers,
		Params:            params,
		PathVars:          pathVars,
		Body:              body,
		Auth:              authB,
		Settings:          settings,
		SortOrder:         int32(p.SortOrder),
		Description:       p.Description,
		SourceSpecID:      pgtype.UUID{Bytes: p.SpecID, Valid: true},
		SourceOperationID: p.OperationID,
		SourceOpHash:      p.OpHash,
		ApiDoc:            apiDoc,
		CreatedBy:         pgtype.UUID{Bytes: p.UserID, Valid: true},
	})
}
