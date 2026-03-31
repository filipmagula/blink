package store

import (
    "database/sql"
    "fmt"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

type FileMeta struct {
    ID              string    `json:"id"`
    Filename        string    `json:"filename"`
    Size            int64     `json:"size"`
    DownloadsLeft   int       `json:"downloads_left"`
    ExpiryTime      time.Time `json:"expiry_time"`
    UploadTime      time.Time `json:"upload_time"`
    UploadedByAdmin bool      `json:"uploaded_by_admin"`
}

type Settings struct {
    MaxDownloads         int `json:"max_downloads"`
    DefaultExpiryM       int `json:"default_expiry_m"`
    MaxAllowedDownloads  int `json:"max_allowed_downloads"`
    MaxAllowedExpiryM    int `json:"max_allowed_expiry_m"`
    MaxAllowedFileSizeMB int `json:"max_allowed_file_size_mb"`
}

type DB struct {
    db *sql.DB
}

func InitDB(path string) (*DB, error) {
    db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
    if err != nil {
        return nil, err
    }

    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS files (
            id TEXT PRIMARY KEY,
            filename TEXT,
            size INTEGER,
            downloads_left INTEGER,
            expiry_time DATETIME,
            upload_time DATETIME,
            uploaded_by_admin INTEGER DEFAULT 0
        );
        CREATE TABLE IF NOT EXISTS settings (
            id INTEGER PRIMARY KEY DEFAULT 1,
            max_downloads INTEGER,
            default_expiry_m INTEGER,
            max_allowed_downloads INTEGER DEFAULT 100,
            max_allowed_expiry_m INTEGER DEFAULT 1440,
            max_allowed_file_size_mb INTEGER DEFAULT 6144
        );
    `)
    if err != nil {
        return nil, err
    }

    // Auto-migrate v1 -> v2 settings
    // If the columns already exist, this will return an error which we safely ignore
    db.Exec("ALTER TABLE settings ADD COLUMN max_allowed_downloads INTEGER DEFAULT 100")
    db.Exec("ALTER TABLE settings ADD COLUMN max_allowed_expiry_m INTEGER DEFAULT 1440")
    db.Exec("ALTER TABLE settings ADD COLUMN max_allowed_file_size_mb INTEGER DEFAULT 6144")
    db.Exec("ALTER TABLE files ADD COLUMN uploaded_by_admin INTEGER DEFAULT 0")
    
    // Ensure no null values if row existed before alter
    db.Exec("UPDATE settings SET max_allowed_downloads = 100 WHERE max_allowed_downloads IS NULL")
    db.Exec("UPDATE settings SET max_allowed_expiry_m = 1440 WHERE max_allowed_expiry_m IS NULL")
    db.Exec("UPDATE settings SET max_allowed_file_size_mb = 6144 WHERE max_allowed_file_size_mb IS NULL")
    db.Exec("UPDATE files SET uploaded_by_admin = 0 WHERE uploaded_by_admin IS NULL")
    
    // Insert default settings if not exists
    db.Exec("INSERT OR IGNORE INTO settings (id, max_downloads, default_expiry_m, max_allowed_downloads, max_allowed_expiry_m, max_allowed_file_size_mb) VALUES (1, 10, 60, 100, 1440, 6144)")
    
    return &DB{db}, nil
}

func (d *DB) Close() error {
    return d.db.Close()
}

func (d *DB) GetSettings() (Settings, error) {
    var s Settings
    err := d.db.QueryRow("SELECT max_downloads, default_expiry_m, max_allowed_downloads, max_allowed_expiry_m, max_allowed_file_size_mb FROM settings WHERE id = 1").
        Scan(&s.MaxDownloads, &s.DefaultExpiryM, &s.MaxAllowedDownloads, &s.MaxAllowedExpiryM, &s.MaxAllowedFileSizeMB)
    return s, err
}

func (d *DB) UpdateSettings(s Settings) error {
    _, err := d.db.Exec("UPDATE settings SET max_downloads = ?, default_expiry_m = ?, max_allowed_downloads = ?, max_allowed_expiry_m = ?, max_allowed_file_size_mb = ? WHERE id = 1",
        s.MaxDownloads, s.DefaultExpiryM, s.MaxAllowedDownloads, s.MaxAllowedExpiryM, s.MaxAllowedFileSizeMB)
    return err
}

func (d *DB) InsertFile(f FileMeta) error {
    _, err := d.db.Exec("INSERT INTO files (id, filename, size, downloads_left, expiry_time, upload_time, uploaded_by_admin) VALUES (?, ?, ?, ?, ?, ?, ?)",
        f.ID, f.Filename, f.Size, f.DownloadsLeft, f.ExpiryTime, f.UploadTime, f.UploadedByAdmin)
    return err
}

// Atomically decrement and return the file metadata if it is valid
func (d *DB) DecrementAndGet(id string) (*FileMeta, error) {
    tx, err := d.db.Begin()
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    var f FileMeta
    err = tx.QueryRow("SELECT id, filename, size, downloads_left, expiry_time, upload_time, uploaded_by_admin FROM files WHERE id = ?", id).
        Scan(&f.ID, &f.Filename, &f.Size, &f.DownloadsLeft, &f.ExpiryTime, &f.UploadTime, &f.UploadedByAdmin)
    
    if err != nil {
        return nil, err // Could be sql.ErrNoRows
    }

    if f.DownloadsLeft <= 0 || time.Now().After(f.ExpiryTime) {
        return nil, fmt.Errorf("expired or depleted")
    }

    _, err = tx.Exec("UPDATE files SET downloads_left = downloads_left - 1 WHERE id = ?", id)
    if err != nil {
        return nil, err
    }
    
    err = tx.Commit()
    if err != nil {
        return nil, err
    }

    return &f, nil
}

func (d *DB) ListFiles() ([]FileMeta, error) {
    rows, err := d.db.Query("SELECT id, filename, size, downloads_left, expiry_time, upload_time, uploaded_by_admin FROM files ORDER BY upload_time DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    files := []FileMeta{}
    for rows.Next() {
        var f FileMeta
        if err := rows.Scan(&f.ID, &f.Filename, &f.Size, &f.DownloadsLeft, &f.ExpiryTime, &f.UploadTime, &f.UploadedByAdmin); err == nil {
            files = append(files, f)
        }
    }
    return files, nil
}

func (d *DB) DeleteFile(id string) error {
    _, err := d.db.Exec("DELETE FROM files WHERE id = ?", id)
    return err
}

func (d *DB) GetStats() (int, int64, error) {
    var count int
    var size sql.NullInt64
    err := d.db.QueryRow("SELECT COUNT(*), SUM(size) FROM files").Scan(&count, &size)
    return count, size.Int64, err
}

func (d *DB) GetExpiredFiles() ([]string, error) {
    rows, err := d.db.Query("SELECT id FROM files WHERE downloads_left <= 0 OR expiry_time <= ?", time.Now())
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    ids := []string{}
    for rows.Next() {
        var id string
        if err := rows.Scan(&id); err == nil {
            ids = append(ids, id)
        }
    }
    return ids, nil
}

func (d *DB) UpdateFileLimits(id string, downloadsLeft int, expiryMinutes int) error {
    newExpiry := time.Now().Add(time.Duration(expiryMinutes) * time.Minute)
    res, err := d.db.Exec("UPDATE files SET downloads_left = ?, expiry_time = ? WHERE id = ? AND uploaded_by_admin = 1",
        downloadsLeft, newExpiry, id)
    if err != nil {
        return err
    }
    affected, err := res.RowsAffected()
    if err != nil {
        return err
    }
    if affected == 0 {
        return fmt.Errorf("file not found or not uploaded by admin")
    }
    return nil
}
