package api

import (
    "encoding/json"
    "net/http"
    "strings"
    "time"
    "os"

    "blink/store"
    "github.com/golang-jwt/jwt/v5"
)

func HandleLogin(expectedPassword, secret string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Password string `json:"password"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Bad request", http.StatusBadRequest)
            return
        }

        if req.Password != expectedPassword {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
            "admin": true,
            "exp":   time.Now().Add(time.Hour * 24).Unix(),
        })

        tokenString, err := token.SignedString([]byte(secret))
        if err != nil {
            http.Error(w, "Token generation failed", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
    }
}

func AuthMiddleware(secret string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
            return []byte(secret), nil
        })

        if err != nil || !token.Valid {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func HandleStats(db *store.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        count, size, err := db.GetStats()
        if err != nil {
            http.Error(w, "Failed to get stats", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "total_files": count,
            "storage_used": size,
            "bandwidth_consumed": 0, // Mock for v1
        })
    }
}

func HandleGetSettings(db *store.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        s, err := db.GetSettings()
        if err != nil {
            http.Error(w, "Failed to get settings", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(s)
    }
}

func HandleUpdateSettings(db *store.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var s store.Settings
        if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
            http.Error(w, "Bad request", http.StatusBadRequest)
            return
        }
        if err := db.UpdateSettings(s); err != nil {
            http.Error(w, "Update failed", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
    }
}

func HandleListFiles(db *store.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        files, err := db.ListFiles()
        if err != nil {
            http.Error(w, "Failed to list files", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(files)
    }
}

func HandleDeleteFile(db *store.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        if id == "" {
            http.Error(w, "Missing ID", http.StatusBadRequest)
            return
        }
        // Delete metadata
        db.DeleteFile(id)
        // Shred payload
        os.Remove("data/uploads/" + id)
        w.WriteHeader(http.StatusOK)
    }
}
