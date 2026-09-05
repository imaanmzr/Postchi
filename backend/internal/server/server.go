package server

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/imaanmzr/postchi/backend/internal/apispec"
	"github.com/imaanmzr/postchi/backend/internal/auth"
	"github.com/imaanmzr/postchi/backend/internal/catalog"
	"github.com/imaanmzr/postchi/backend/internal/collab"
	"github.com/imaanmzr/postchi/backend/internal/collection"
	"github.com/imaanmzr/postchi/backend/internal/db"
	"github.com/imaanmzr/postchi/backend/internal/docsync"
	"github.com/imaanmzr/postchi/backend/internal/environment"
	"github.com/imaanmzr/postchi/backend/internal/history"
	"github.com/imaanmzr/postchi/backend/internal/importexport"
	"github.com/imaanmzr/postchi/backend/internal/invite"
	"github.com/imaanmzr/postchi/backend/internal/publicconfig"
	"github.com/imaanmzr/postchi/backend/internal/request"
	"github.com/imaanmzr/postchi/backend/internal/share"
	"github.com/imaanmzr/postchi/backend/internal/shared/config"
	"github.com/imaanmzr/postchi/backend/internal/shared/crypto"
	appMiddleware "github.com/imaanmzr/postchi/backend/internal/shared/middleware"
	"github.com/imaanmzr/postchi/backend/internal/workspace"
	"github.com/imaanmzr/postchi/backend/internal/workspacetoken"
)

type Server struct {
	http *http.Server
}

func New(cfg *config.Config, log *zap.Logger, pool *pgxpool.Pool) *Server {
	cryptoSvc, err := crypto.NewService(cfg.EncryptionKey)
	if err != nil {
		log.Fatal("invalid encryption key (must be 32 bytes)", zap.Error(err))
	}

	tokens := auth.NewService(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	store := db.NewStore(pool)
	authH := auth.NewHandler(store, tokens, cfg)
	wsH := workspace.NewHandler(store)
	colH := collection.NewHandler(store)
	reqH := request.NewHandler(store, cfg, cryptoSvc)
	envH := environment.NewHandler(store, cryptoSvc)
	histH := history.NewHandler(store)
	impH := importexport.NewHandler(store, cryptoSvc)
	inviteH := invite.NewHandler(store, cfg, tokens)
	shareH := share.NewHandler(store, cfg)
	apispecH := apispec.NewHandler(store, cfg)
	catalogH := catalog.NewHandler(store)
	docsyncH := docsync.NewHandler(store, cryptoSvc)
	wsTokenH := workspacetoken.NewHandler(store)
	publicCfgH := publicconfig.NewHandler(cfg)
	hub := collab.NewHub(log)

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Recoverer)
	r.Use(appMiddleware.Logger(log))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", appMiddleware.Health)
	r.Get("/ready", appMiddleware.Ready(func(ctx context.Context) error { return pool.Ping(ctx) }))

	r.Route("/api", func(r chi.Router) {
		r.Get("/config/public", publicCfgH.Get)

		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/refresh", authH.Refresh)
		r.Post("/auth/logout", authH.Logout)
		r.Post("/auth/forgot-password", authH.ForgotPassword)
		r.Get("/auth/reset-password/{token}", authH.PreviewResetPassword)
		r.Post("/auth/reset-password/{token}", authH.ResetPassword)

		r.Get("/invites/{token}", inviteH.Preview)
		r.Post("/invites/{token}/accept", inviteH.Accept)
		r.Get("/shares/{token}", shareH.GetByToken)

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(tokens))

			r.Get("/auth/me", authH.Me)
			r.Post("/auth/change-password", authH.ChangePassword)

			r.Get("/workspaces", wsH.List)
			r.Post("/workspaces", wsH.Create)
			r.Route("/workspaces/{id}", func(r chi.Router) {
				r.With(wsH.RequireRole("viewer")).Get("/", wsH.Get)
				r.With(wsH.RequireRole("editor")).Patch("/", wsH.Update)
				r.With(wsH.RequireRole("owner")).Delete("/", wsH.Delete)
				r.With(wsH.RequireRole("viewer")).Get("/members", wsH.ListMembers)
				r.With(wsH.RequireRole("owner")).Post("/members", wsH.AddMember)
				r.With(wsH.RequireRole("owner")).Patch("/members/{userId}", wsH.UpdateMember)
				r.With(wsH.RequireRole("owner")).Delete("/members/{userId}", wsH.RemoveMember)
				r.With(wsH.RequireRole("owner")).Get("/invites", inviteH.List)
				r.With(wsH.RequireRole("owner")).Post("/invites", inviteH.Create)
				r.With(wsH.RequireRole("owner")).Delete("/invites/{inviteId}", inviteH.Revoke)
				r.With(wsH.RequireRole("viewer")).Get("/activity", wsH.Activity)
				r.With(wsH.RequireRole("viewer")).Get("/collections", colH.ListByWorkspace)
				r.With(wsH.RequireRole("viewer")).Get("/requests", reqH.ListByWorkspace)
				r.With(wsH.RequireRole("editor")).Post("/requests/backfill-operation-ids", reqH.BackfillOperationIDs)
				r.With(wsH.RequireRole("viewer")).Get("/shares", shareH.List)
				r.With(wsH.RequireRole("viewer")).Get("/catalog", catalogH.WorkspaceCatalog)
				r.With(wsH.RequireRole("viewer")).Get("/workspace-docs", docsyncH.ListDocs)
				r.With(wsH.RequireRole("editor")).Post("/workspace-docs", docsyncH.CreateDoc)
				r.With(wsH.RequireRole("viewer")).Get("/workspace-docs/{docId}/request-links", docsyncH.ListDocRequestLinks)
				r.With(wsH.RequireRole("viewer")).Get("/workspace-docs/{docId}/links", docsyncH.ListDocLinks)
				r.With(wsH.RequireRole("editor")).Post("/workspace-docs/{docId}/links", docsyncH.CreateDocLink)
				r.With(wsH.RequireRole("editor")).Delete("/workspace-docs/{docId}/links/{linkId}", docsyncH.DeleteDocLink)
				r.With(wsH.RequireRole("viewer")).Get("/workspace-docs/{slug}", docsyncH.GetDoc)
				r.With(wsH.RequireRole("editor")).Patch("/workspace-docs/{slug}", docsyncH.UpdateDoc)
				r.With(wsH.RequireRole("editor")).Delete("/workspace-docs/{slug}", docsyncH.DeleteDoc)
				r.With(wsH.RequireRole("viewer")).Get("/doc-graph", docsyncH.GetDocGraph)
				r.With(wsH.RequireRole("editor")).Post("/doc-links/analyze", docsyncH.AnalyzeDocLinks)
				r.With(wsH.RequireRole("viewer")).Get("/doc-links/suggestions", docsyncH.ListDocLinkSuggestions)
				r.With(wsH.RequireRole("editor")).Post("/doc-links/suggestions/accept-all", docsyncH.AcceptAllDocLinkSuggestions)
				r.With(wsH.RequireRole("editor")).Post("/doc-links/suggestions/{suggestionId}/accept", docsyncH.AcceptDocLinkSuggestion)
				r.With(wsH.RequireRole("editor")).Post("/doc-links/suggestions/{suggestionId}/reject", docsyncH.RejectDocLinkSuggestion)
				r.With(wsH.RequireRole("viewer")).Get("/doc-sources", docsyncH.ListSources)
				r.With(wsH.RequireRole("viewer")).Get("/doc-sources/{sourceId}/branches", docsyncH.ListSourceBranches)
				r.With(wsH.RequireRole("editor")).Post("/doc-sources", docsyncH.CreateSource)
				r.With(wsH.RequireRole("editor")).Patch("/doc-sources/{sourceId}", docsyncH.UpdateSource)
				r.With(wsH.RequireRole("editor")).Delete("/doc-sources/{sourceId}", docsyncH.DeleteSource)
				r.With(wsH.RequireRole("editor")).Post("/git/branches/preview", docsyncH.PreviewBranches)
				r.With(wsH.RequireRole("owner")).Get("/api-tokens", wsTokenH.List)
				r.With(wsH.RequireRole("owner")).Post("/api-tokens", wsTokenH.Create)
				r.With(wsH.RequireRole("owner")).Delete("/api-tokens/{tokenId}", wsTokenH.Revoke)
				r.With(wsH.RequireRole("editor")).Post("/api-specs/upload", apispecH.Upload)
				r.With(wsTokenH.RequireScope("spec:push")).Post("/api-specs/push", apispecH.Push)
				r.With(wsH.RequireRole("viewer")).Get("/api-specs", apispecH.List)
				r.With(wsH.RequireRole("editor")).Post("/api-specs", apispecH.Create)
				r.With(wsH.RequireRole("viewer")).Get("/bruno-sources", impH.ListBrunoSources)
				r.With(wsH.RequireRole("viewer")).Get("/bruno-sources/{sourceId}/branches", impH.ListBrunoSourceBranches)
				r.With(wsH.RequireRole("editor")).Post("/bruno-sources", impH.CreateBrunoSource)
				r.With(wsH.RequireRole("editor")).Patch("/bruno-sources/{sourceId}", impH.UpdateBrunoSource)
				r.With(wsH.RequireRole("editor")).Delete("/bruno-sources/{sourceId}", impH.DeleteBrunoSource)
				r.With(wsH.RequireRole("viewer")).Get("/collection-sources", impH.ListBrunoSources)
				r.With(wsH.RequireRole("viewer")).Get("/collection-sources/{sourceId}/branches", impH.ListBrunoSourceBranches)
				r.With(wsH.RequireRole("editor")).Post("/collection-sources", impH.CreateBrunoSource)
				r.With(wsH.RequireRole("editor")).Patch("/collection-sources/{sourceId}", impH.UpdateBrunoSource)
				r.With(wsH.RequireRole("editor")).Delete("/collection-sources/{sourceId}", impH.DeleteBrunoSource)
				r.With(wsH.RequireRole("editor")).Post("/imports/bruno/git", impH.ImportBrunoGit)
				r.With(wsH.RequireRole("editor")).Post("/imports/git", impH.ImportCollectionGit)
			})

			r.Post("/collections", colH.Create)
			r.Patch("/collections/reorder", colH.Reorder)
			r.Route("/collections/{id}", func(r chi.Router) {
				r.Get("/", colH.Get)
				r.Patch("/", colH.Update)
				r.Delete("/", colH.Delete)
				r.Post("/duplicate", colH.Duplicate)
				r.Get("/docs", colH.Docs)
				r.Get("/catalog", catalogH.CollectionCatalog)
				r.Post("/run", reqH.RunCollection)
			})

			r.Get("/requests", reqH.ListByCollection)
			r.Post("/requests", reqH.Create)
			r.Patch("/requests/reorder", reqH.Reorder)
			r.Route("/requests/{id}", func(r chi.Router) {
				r.Get("/", reqH.Get)
				r.Get("/docs-bundle", reqH.GetDocsBundle)
				r.Patch("/", reqH.Update)
				r.Delete("/", reqH.Delete)
				r.Patch("/move", reqH.Move)
				r.Post("/execute", reqH.Execute)
				r.Post("/duplicate", reqH.Duplicate)
				r.Get("/snippet", reqH.Snippet)
				r.Post("/examples", reqH.SaveExample)
				r.Post("/children", reqH.CreateChild)
				r.Get("/children", reqH.ListChildren)
				r.Post("/reset-field", reqH.ResetField)
				r.Post("/promote-to-template", reqH.PromoteToTemplate)
				r.Post("/push-to-children", reqH.PushToChildren)
			})

			r.Post("/shares", shareH.Create)
			r.Delete("/shares/{id}", shareH.Revoke)
			r.Post("/shares/{token}/import", shareH.Import)

			r.Route("/api-specs/{id}", func(r chi.Router) {
				r.Get("/", apispecH.Get)
				r.Patch("/", apispecH.Update)
				r.Delete("/", apispecH.Delete)
				r.Put("/environment-urls", apispecH.SetEnvironmentURLs)
				r.Post("/sync", apispecH.Sync)
				r.Post("/reupload", apispecH.Reupload)
			})

			r.Post("/doc-sources/{id}/sync", docsyncH.SyncSource)
			r.Post("/bruno-sources/{id}/sync", impH.SyncBrunoSource)
			r.Post("/collection-sources/{id}/sync", impH.SyncCollectionSource)

			r.Get("/environments", envH.List)
			r.Post("/environments", envH.Create)
			r.Route("/environments/{id}", func(r chi.Router) {
				r.Get("/", envH.Get)
				r.Patch("/", envH.Update)
				r.Delete("/", envH.Delete)
				r.Post("/resolve-variables", envH.ResolveVariables)
				r.Post("/variables/bulk", envH.BulkSetVariables)
			})

			r.Get("/history", histH.List)

			r.Post("/import/postman", impH.ImportPostman)
			r.Post("/import/openapi", impH.ImportOpenAPI)
			r.Post("/import/opencollection", impH.ImportOpenCollection)
			r.Post("/import/curl", impH.ImportCurl)
			r.Post("/import/bruno", impH.ImportBruno)
			r.Get("/export/postman", impH.ExportPostman)
			r.Get("/export/bruno", impH.ExportBruno)

			r.Get("/ws", hub.HandleWS)
		})
	})

	if cfg.StaticFilesPath != "" {
		r.Handle("/*", spaHandler(cfg.StaticFilesPath))
	}

	return &Server{
		http: &http.Server{
			Addr:         ":" + cfg.HTTPPort,
			Handler:      r,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 120 * time.Second,
		},
	}
}

func spaHandler(staticDir string) http.Handler {
	fs := http.FileServer(http.Dir(staticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(staticDir, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
