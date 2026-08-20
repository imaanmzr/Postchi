package importexport

import (
	"sync"

	"github.com/google/uuid"
)

func (h *Handler) sourceSyncMutex(sourceID uuid.UUID) *sync.Mutex {
	value, _ := h.sourceSyncMu.LoadOrStore(sourceID.String(), &sync.Mutex{})
	return value.(*sync.Mutex)
}
