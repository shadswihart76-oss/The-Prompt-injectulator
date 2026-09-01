package main

import (
	"log"
	"net/http"
	"os"

	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/auth"
	"github.com/shadswihart76-oss/the-prompt-injectulator/internal/handler"
)

func main() {
	cfg, err := auth.LoadConfig()
	if err != nil {
		log.Fatalf("configuration error: %v\n(Hint: set DEV_MODE=true for local development)", err)
	}

	if cfg.DevMode {
		log.Println("⚠️  DEV_MODE is ON – test account active, mock provider available. Do NOT use in production.")
		log.Printf("    Dev test email: %s", cfg.DevTestEmail)
	}

	store := auth.NewStore()
	srv := handler.NewServer(cfg, store)

	mux := http.NewServeMux()

	// Auth endpoints
	mux.HandleFunc("POST /api/auth/register", srv.Register)
	mux.HandleFunc("POST /api/auth/login", srv.Login)

	// LLM completion (authenticated)
	mux.HandleFunc("POST /api/llm/complete", srv.Complete)

	// Usage endpoints (authenticated)
	mux.HandleFunc("GET /api/usage", srv.UsageStatus)

	// Admin endpoints
	mux.HandleFunc("GET /api/admin/status", srv.AdminStatus)
	mux.HandleFunc("POST /api/admin/reset-usage", srv.ResetUsage)

	// Serve static frontend files if present.
	if _, err := os.Stat("./static"); err == nil {
		mux.Handle("/", http.FileServer(http.Dir("./static")))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"Prompt Injectulator","status":"running"}`)) //nolint:errcheck
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Prompt Injectulator on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
