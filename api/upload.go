package api

import (
    "context"
    "errors"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "time"

    "blink/store"
    "github.com/matoous/go-nanoid/v2"
)

func handleUploadInternal(db *store.DB, w http.ResponseWriter, r *http.Request, isAdmin bool) {
    settings, _ := db.GetSettings()
    maxBytes := int64(1024 * 1024 * 1024)
    if settings.MaxAllowedFileSizeMB > 0 {
        maxBytes = int64(settings.MaxAllowedFileSizeMB) * 1024 * 1024
    }
    
    // Enforce max body size length
    r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

    id, err := gonanoid.New(16)
    if err != nil {
        http.Error(w, "Internal generate error", http.StatusInternalServerError)
        return
    }

    if settings.MaxDownloads == 0 { settings.MaxDownloads = 1; settings.DefaultExpiryM = 60 }
    if settings.MaxAllowedDownloads == 0 { settings.MaxAllowedDownloads = 100 }
    if settings.MaxAllowedExpiryM == 0 { settings.MaxAllowedExpiryM = 1440 }

    maxDownloads := settings.MaxDownloads
    if hdl := r.Header.Get("Max-Downloads"); hdl != "" {
        if val, err := strconv.Atoi(hdl); err == nil && val > 0 {
            maxDownloads = val
        }
    }
    if !isAdmin && maxDownloads > settings.MaxAllowedDownloads {
        maxDownloads = settings.MaxAllowedDownloads
    }

    expiryMinutes := settings.DefaultExpiryM
    if expStr := r.Header.Get("Expiry"); expStr != "" {
        if duration, err := time.ParseDuration(expStr); err == nil {
            if m := int(duration.Minutes()); m > 0 {
                expiryMinutes = m
            }
        } else if m, err := strconv.Atoi(expStr); err == nil && m > 0 {
            expiryMinutes = m
        }
    }
    if !isAdmin && expiryMinutes > settings.MaxAllowedExpiryM {
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

        var maxBytesErr *http.MaxBytesError
        switch {
        case errors.As(err, &maxBytesErr):
            http.Error(w, "Payload too large", http.StatusRequestEntityTooLarge)
        case errors.Is(err, context.Canceled):
            // Client canceled upload; connection is gone.
        default:
            http.Error(w, "Upload failed", http.StatusBadRequest)
        }
        return
    }

    meta := store.FileMeta{
        ID:              id,
        Filename:        filename,
        Size:            written,
        DownloadsLeft:   maxDownloads,
        UploadTime:      time.Now(),
        ExpiryTime:      time.Now().Add(time.Duration(expiryMinutes) * time.Minute),
        UploadedByAdmin: isAdmin,
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

func HandleUpload(db *store.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        handleUploadInternal(db, w, r, false)
    }
}

func HandleUploadWithFilename(db *store.DB) http.HandlerFunc {
    return HandleUpload(db)
}

func HandleAdminUpload(db *store.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        handleUploadInternal(db, w, r, true)
    }
}
