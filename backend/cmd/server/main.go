package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"heatwave/backend/internal/api"
	"heatwave/backend/internal/config"
	"heatwave/backend/internal/orchestration"
)

func main() {
	cfg := config.Load()

	orch, err := orchestration.New(cfg)
	if err != nil {
		log.Fatalf("orchestration init: %v", err)
	}

	if cfg.AutoRun {
		go orch.RunPipelineLoop(cfg.PipelineInterval)
	}

	apiMux := api.NewRouter(cfg, orch)

	// Wrap: API + frontend static (built dist/)
	root := http.NewServeMux()
	root.Handle("/api/", apiMux)
	root.Handle("/api", apiMux)

	staticDir := findDist()
	if staticDir != "" {
		log.Printf("serving frontend from %s", staticDir)
		fs := http.FileServer(http.Dir(staticDir))
		root.Handle("/data/", fs)
		root.Handle("/assets/", immutable(staticDir))
		root.Handle("/", spaHandler(staticDir))
	} else {
		root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend dist/ not built. Run: cd frontend && npm run build", http.StatusNotFound)
		})
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           withCORS(root),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("HEEWS backend listening on :%s mode=%s", cfg.Port, cfg.AppMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("shutdown ok")
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func immutable(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
	})
}

func spaHandler(dir string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, r.URL.Path)
		if fi, err := os.Stat(path); err != nil || fi.IsDir() {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			http.ServeFile(w, r, index)
			return
		}
		fs.ServeHTTP(w, r)
	}
}

func findDist() string {
	candidates := []string{
		"frontend/dist",
		"../frontend/dist",
		"dist",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "index.html")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}