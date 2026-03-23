package api

import (
    "net/http"
    "os"
    "path/filepath"
    "blink/store"
)

func HandleDownload(db *store.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        if id == "" || id == "admin" {
            http.NotFound(w, r)
            return
        }

        // Prevent directory traversal attacks
        if filepath.Base(id) != id {
            http.NotFound(w, r)
            return
        }

        // Atomically decrement
        f, err := db.DecrementAndGet(id)
        if err != nil {
            // File expired, depleted, or doesn't exist
            // Return generic 404 to avoid information disclosure
            http.NotFound(w, r)
            return
        }

        dstPath := filepath.Join("data", "uploads", id)
        fileInfo, err := os.Stat(dstPath)
        if err != nil || fileInfo.IsDir() {
            http.NotFound(w, r)
            return
        }

        w.Header().Set("Content-Disposition", `attachment; filename="`+f.Filename+`"`)
        w.Header().Set("Content-Type", "application/octet-stream")
        
        // Let stdlib efficiently stream file buffers to the response
        http.ServeFile(w, r, dstPath)
    }
}
