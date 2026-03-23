package api

import (
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "time"

    "blink/store"
    "github.com/matoous/go-nanoid/v2"
)

func HandleUpload(db *store.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Enforce 1GB max body size length
        r.Body = http.MaxBytesReader(w, r.Body, 1024*1024*1024)

        id, err := gonanoid.New(16)
        if err != nil {
            http.Error(w, "Internal generate error", http.StatusInternalServerError)
            return
        }

        settings, _ := db.GetSettings()
        if settings.MaxDownloads == 0 { settings.MaxDownloads = 1; settings.DefaultExpiryM = 60 }
        if settings.MaxAllowedDownloads == 0 { settings.MaxAllowedDownloads = 100 }
        if settings.MaxAllowedExpiryM == 0 { settings.MaxAllowedExpiryM = 1440 }

        maxDownloads := settings.MaxDownloads
        if hdl := r.Header.Get("Max-Downloads"); hdl != "" {
            if val, err := strconv.Atoi(hdl); err == nil && val > 0 {
                maxDownloads = val
            }
        }
        if maxDownloads > settings.MaxAllowedDownloads {
            maxDownloads = settings.MaxAllowedDownloads
        }

        expiryMinutes := settings.DefaultExpiryM
        if expStr := r.Header.Get("Expiry"); expStr != "" {
            if duration, err := time.ParseDuration(expStr); err == nil {
                if m := int(duration.Minutes()); m > 0 {
                    expiryMinutes = m
                }
            }
        }
        if expiryMinutes > settings.MaxAllowedExpiryM {
            expiryMinutes = settings.MaxAllowedExpiryM
        }

        filename := r.PathValue("filename")
        if filename == "" || filename == "/" {
            filename = "secret_file"
        }

        dstPath := filepath.Join("data", "uploads", id)
        f, err := os.Create(dstPath)
        if err != nil {
            http.Error(w, "Storage unavailable", http.StatusInternalServerError)
            return
        }
        
        written, err := io.Copy(f, r.Body)
        f.Close() // Close immediately to flush
        
        if err != nil {
            os.Remove(dstPath)
            http.Error(w, "Upload failed or exceeded maximum payload", http.StatusBadRequest)
            return
        }

        meta := store.FileMeta{
            ID:            id,
            Filename:      filename,
            Size:          written,
            DownloadsLeft: maxDownloads,
            UploadTime:    time.Now(),
            ExpiryTime:    time.Now().Add(time.Duration(expiryMinutes) * time.Minute),
        }

        if err := db.InsertFile(meta); err != nil {
            os.Remove(dstPath)
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }

        scheme := "http"
        if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" { scheme = "https" }
        w.Header().Set("Content-Type", "text/plain")

        responseMsg := scheme + "://" + r.Host + "/" + id + "\n"
        responseMsg += "Downloads allowed: " + strconv.Itoa(maxDownloads) + "\n"
        responseMsg += "Expires in: " + strconv.Itoa(expiryMinutes) + " minutes\n"
        
        w.Write([]byte(responseMsg))
    }
}

func HandleUploadWithFilename(db *store.DB) http.HandlerFunc {
    return HandleUpload(db)
}
