package request

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/db/sqlc"
	"github.com/imaanmzr/postchi/backend/internal/shared/operationid"
	"github.com/imaanmzr/postchi/backend/internal/shared/respond"
)

func (h *Handler) BackfillOperationIDs(w http.ResponseWriter, r *http.Request) {
	wsID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid workspace id")
		return
	}
	ctx := r.Context()
	rows, err := h.store.ListRequestsForOperationBackfill(ctx, db.PGUUID(wsID))
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "query failed")
		return
	}
	var updated, skipped int
	for _, row := range rows {
		opID := operationid.CanonicalFromMethodURL(row.Method, row.Url)
		if opID == "" {
			skipped++
			continue
		}
		n, err := h.store.BackfillRequestOperationIDs(ctx, sqlc.BackfillRequestOperationIDsParams{
			WorkspaceID:       db.PGUUID(wsID),
			RequestID:         row.ID,
			SourceOperationID: opID,
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "update failed")
			return
		}
		if n > 0 {
			updated++
		} else {
			skipped++
		}
	}
	respond.JSON(w, http.StatusOK, map[string]int{
		"updated": updated,
		"skipped": skipped,
	})
}
