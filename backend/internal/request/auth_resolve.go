package request

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/imaanmzr/postchi/backend/internal/db"
)

func explicitAuth(a AuthSpec) bool {
	return a.Type != "" && a.Type != "inherit"
}

// ResolveAuthChain resolves auth from a request spec and collection auths ordered leaf-to-root.
func ResolveAuthChain(reqAuth AuthSpec, collectionAuthsLeafToRoot []AuthSpec) AuthSpec {
	if explicitAuth(reqAuth) {
		return reqAuth
	}
	for _, a := range collectionAuthsLeafToRoot {
		if explicitAuth(a) {
			return a
		}
	}
	return AuthSpec{Type: "none"}
}

func (e *Executor) ResolveAuth(ctx context.Context, collectionID string, auth AuthSpec) AuthSpec {
	if explicitAuth(auth) {
		return auth
	}
	cid, err := uuid.Parse(collectionID)
	if err != nil {
		return AuthSpec{Type: "none"}
	}
	chain := e.collectionAncestorIDs(ctx, cid)
	auths := make([]AuthSpec, 0, len(chain))
	for i := len(chain) - 1; i >= 0; i-- {
		authB, err := e.store.GetCollectionAuth(ctx, db.PGUUID(chain[i]))
		if err != nil {
			continue
		}
		var colAuth AuthSpec
		_ = json.Unmarshal(authB, &colAuth)
		auths = append(auths, colAuth)
	}
	return ResolveAuthChain(auth, auths)
}
