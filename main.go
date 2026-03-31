package main

import (
    "embed"
    "io/fs"
    "log"
    "net/http"
    "os"

    "blink/api"
    "blink/store"
    "blink/worker"
)

//go:embed frontend/dist/*
var frontendDist embed.FS

func main() {
    // Ensure data directories exist
    os.MkdirAll("data/uploads", 0777)

    // Init DB
    dbPath := "data/sqlite.db"
    db, err := store.InitDB(dbPath)
    if err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer db.Close() // In practice, server runs infinitely

    // Start background cleanup worker
    go worker.StartCleanupWorker(db)

    // Setup HTTP multiplexer
    mux := http.NewServeMux()

    // API Handlers
    appToken := os.Getenv("JWT_SECRET")
    if appToken == "" {
        appToken = "change-me-in-production"
    }

    adminPassword := os.Getenv("ADMIN_PASSWORD")
    if adminPassword == "" {
        adminPassword = "supersecretadmin"
    }

    mux.HandleFunc("PUT /", api.HandleUpload(db))
    mux.HandleFunc("PUT /{filename}", api.HandleUploadWithFilename(db))
    mux.HandleFunc("GET /{id}", api.HandleDownload(db))
    
    // Auth & Admin APIs
    mux.HandleFunc("POST /admin/api/login", api.HandleLogin(adminPassword, appToken))
    
    // Protected Admin Routes via middleware
    mux.Handle("GET /admin/api/stats", api.AuthMiddleware(appToken, api.HandleStats(db)))
    mux.Handle("GET /admin/api/settings", api.AuthMiddleware(appToken, api.HandleGetSettings(db)))
    mux.Handle("PUT /admin/api/settings", api.AuthMiddleware(appToken, api.HandleUpdateSettings(db)))
    mux.Handle("GET /admin/api/files", api.AuthMiddleware(appToken, api.HandleListFiles(db)))
    mux.Handle("DELETE /admin/api/files/{id}", api.AuthMiddleware(appToken, api.HandleDeleteFile(db)))
    mux.Handle("PUT /admin/api/files/{id}", api.AuthMiddleware(appToken, api.HandleUpdateFile(db)))
    mux.Handle("POST /admin/api/upload", api.AuthMiddleware(appToken, api.HandleAdminUpload(db)))
    mux.Handle("POST /admin/api/upload/{filename}", api.AuthMiddleware(appToken, api.HandleAdminUpload(db)))

    // Sub direct embedded dist folder (stripped of frontend/dist)
    dist, err := fs.Sub(frontendDist, "frontend/dist")
    if err != nil {
        // Do not crash locally if building without frontend for testing
        log.Printf("Warning: embedded frontend/dist not found. Admin UI will be unavailable if not correctly built.")
    } else {
        // Serve embedded Vue admin app at /admin/
        mux.Handle("GET /admin/", http.StripPrefix("/admin/", http.FileServer(http.FS(dist))))
        
        // Redirect /admin to /admin/ for better UX
        mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
            http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
        })
    }

    // Explicit 404 for root GET to prevent indexing leaking files
    mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
        http.NotFound(w, r)
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "21337"
    }
    
    log.Printf("Blink server running on :%s", port)
    if err := http.ListenAndServe(":"+port, mux); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
