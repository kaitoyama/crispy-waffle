// Command server runs the graph-based task/workflow backend: it opens the
// SQLite store, wires the workflow engine, seeds the reference workflow, and
// serves the REST API (and the built frontend if present).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kaitoyama/crispy-waffle/internal/api"
	"github.com/kaitoyama/crispy-waffle/internal/exec"
	"github.com/kaitoyama/crispy-waffle/internal/seed"
	"github.com/kaitoyama/crispy-waffle/internal/service"
	"github.com/kaitoyama/crispy-waffle/internal/store"
	"github.com/kaitoyama/crispy-waffle/internal/tools"
	"github.com/kaitoyama/crispy-waffle/internal/tools/builtin"
	"github.com/kaitoyama/crispy-waffle/internal/workflow"
)

func main() {
	addr := getenv("ADDR", ":8080")
	dbPath := getenv("DB_PATH", "./data/app.db")

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()
	log.Printf("store: migrated %s", dbPath)

	toolRegistry := tools.NewRegistry()
	builtin.Register(toolRegistry)

	registry := workflow.NewRegistry()
	engine := &workflow.Engine{
		Events: st, Runs: st, Tasks: st, Actors: st,
		Registry: registry, Executor: exec.NewExecutor(toolRegistry),
	}

	svc := service.New(st, engine, registry, seed.ActorAccBot)
	svc.Tools = toolRegistry

	ctx := context.Background()
	if err := seed.Install(ctx, st, registry, svc); err != nil {
		log.Fatalf("seed: %v", err)
	}

	srv := &api.Server{Svc: svc, Tools: toolRegistry, Registry: registry}

	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Handler())
	// Serve the built frontend if it exists (production); in dev, Vite proxies.
	if _, err := os.Stat("./frontend/dist"); err == nil {
		mux.Handle("/", spaHandler("./frontend/dist"))
		log.Printf("serving frontend from ./frontend/dist")
	}

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("listening %s", addr)
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// spaHandler serves static files and falls back to index.html for client-side
// routes (so deep links work).
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(dir + r.URL.Path); os.IsNotExist(err) && r.URL.Path != "/" {
			http.ServeFile(w, r, dir+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
