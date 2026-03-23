package worker

import (
    "log"
    "os"
    "path/filepath"
    "time"

    "blink/store"
)

func StartCleanupWorker(db *store.DB) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        ids, err := db.GetExpiredFiles()
        if err != nil {
            log.Printf("Background worker failed to fetch expired metadata: %v", err)
            continue
        }

        for _, id := range ids {
            // Delete DB record first to prevent inflight downloads
            if err := db.DeleteFile(id); err != nil {
                continue
            }
            // Unlink underlying file structure
            err := os.Remove(filepath.Join("data", "uploads", id))
            if err == nil {
                log.Printf("Successfully purged inactive payload: %s", id)
            }
        }
    }
}
