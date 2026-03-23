package store

import (
    "database/sql"
    "fmt"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

type FileMeta struct {
    ID            string    `json:"id"`
    Filename      string    `json:"filename"`
    Size          int64     `json:"size"`
    DownloadsLeft int       `json:"downloads_left"`
    ExpiryTime    time.Time `json:"expiry_time"`
    UploadTime    time.Time `json:"upload_time"`
}

type Settings struct {
    MaxDownloads   int `json:"max_downloads"`
    DefaultExpiryM int `json:"default_expiry_m"`
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
            upload_time DATETIME
        );
        CREATE TABLE IF NOT EXISTS settings (
            id INTEGER PRIMARY KEY DEFAULT 1,
            max_downloads INTEGER,
            default_expiry_m INTEGER
        );
        INSERT OR IGNORE INTO settings (id, max_downloads, default_expiry_m) VALUES (1, 10, 60);
    `)
    if err != nil {
        return nil, err
    }
    return &DB{db}, nil
}

func (d *DB) GetSettings() (Settings, error) {
    var s Settings
    err := d.db.QueryRow("SELECT max_downloads, default_expiry_m FROM settings WHERE id = 1").Scan(&s.MaxDownloads, &s.DefaultExpiryM)
    return s, err
}

func (d *DB) UpdateSettings(s Settings) error {
    _, err := d.db.Exec("UPDATE settings SET max_downloads = ?, default_expiry_m = ? WHERE id = 1", s.MaxDownloads, s.DefaultExpiryM)
    return err
}

func (d *DB) InsertFile(f FileMeta) error {
    _, err := d.db.Exec("INSERT INTO files (id, filename, size, downloads_left, expiry_time, upload_time) VALUES (?, ?, ?, ?, ?, ?)",
        f.ID, f.Filename, f.Size, f.DownloadsLeft, f.ExpiryTime, f.UploadTime)
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
    err = tx.QueryRow("SELECT id, filename, size, downloads_left, expiry_time, upload_time FROM files WHERE id = ?", id).
        Scan(&f.ID, &f.Filename, &f.Size, &f.DownloadsLeft, &f.ExpiryTime, &f.UploadTime)
    
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
    rows, err := d.db.Query("SELECT id, filename, size, downloads_left, expiry_time, upload_time FROM files ORDER BY upload_time DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    files := []FileMeta{}
    for rows.Next() {
        var f FileMeta
        if err := rows.Scan(&f.ID, &f.Filename, &f.Size, &f.DownloadsLeft, &f.ExpiryTime, &f.UploadTime); err == nil {
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
