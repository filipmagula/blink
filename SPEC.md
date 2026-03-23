# Blink - Secure-Transfer Service Specification

## 1. Core Architecture
- **Language**: Go (Golang) - Highly performant, statically typed, minimal memory footprint, and provides excellent concurrency primitives (`goroutines`) for networked services.
- **Metadata Store**: SQLite - Highly recommended over an in-memory map. SQLite (with WAL mode enabled) provides ACID guarantees ensuring atomic decrement of downloads, persists data across server restarts, and keeps memory usage low. 
- **Storage**: Flat-file system. Files are stored using high-entropy generated IDs (e.g., NanoID or UUIDv4) rather than their original names to prevent directory traversal or ID guessing.

## 2. API Design (CLI Focused)

### 2.1. File Upload

**Proposed Better Option:**
When using `curl --upload-file` (or `-T`) against a URL terminating in a slash, `curl` automatically appends the local filename to the URL. This performs an HTTP `PUT` request with the file name natively included in the path, allowing the server to capture the original filename without needing complex `multipart/form-data` parsing.

*Command:*
```bash
curl --upload-file ./secret.enc \
     -H "Max-Downloads: 2" \
     -H "Expiry: 10m" \
     https://your-vps.com/
```

*Server Behavior:*
1. Receives `PUT /secret.enc`.
2. Extracts `secret.enc` as the original filename.
3. Generates a cryptic ID (e.g., NanoID `a1b2c3d4Lx9`).
4. Streams the request body directly to disk as `a1b2c3d4Lx9`.
5. Returns the unique download URL: `https://your-vps.com/a1b2c3d4Lx9`

### 2.2. File Download

**Proposed Better Option:**
By returning a `Content-Disposition` header, the server can instruct the downloader's CLI tool to save the file under its original name, eliminating the need for the user to manually pipe the output via `> secret.enc`.

*Command:*
```bash
# -O: Write output to a local file
# -J: Use the server-provided filename from the Content-Disposition header
curl -O -J https://your-vps.com/a1b2c3d4Lx9
```

*Server Behavior:*
1. Locates the metadata for `a1b2c3d4Lx9`.
2. Validates download limits and expiry time.
3. Atomically decrements the download count in SQLite.
4. Responds with `Content-Disposition: attachment; filename="secret.enc"`.
5. Streams the file payload to the client.

## 3. Security Measures

1. **Header Validation**: 
   - Parse `Max-Downloads` (Limit: 1-10) and `Expiry` (Limit: 1m - 1h).
   - Impose strict defaults if headers are missing or malformed (e.g., 1 download, 10m expiry).
2. **Atomic Increments/Decrements**: 
   - Utilize SQLite transactions paired with the `RETURNING` clause to decrement counts safely in one query. 
   - *Example:* `UPDATE files SET downloads_left = downloads_left - 1 WHERE id = ? AND downloads_left > 0 RETURNING downloads_left;`
   - This explicitly prevents race conditions where simultaneous requests exceed the maximum allowed downloads.
3. **Zero Information Disclosure (No Indexing)**:
   - The server root (`/`) returns a generic `404 Not Found`.
   - Missing, expired, or fully depleted files seamlessly return the same generic `404 Not Found`. Never use `401 Unauthorized` or `410 Gone`, as doing so could leak the existence of an uploaded file to an active scanner.
4. **Automated Cleanup Worker**:
   - A background Goroutine running a `time.Ticker` (e.g., every 1 minute).
   - Scans SQLite for expired records or deplete-download bounds, unlinks (deletes) the physical file from the storage system, and drops the record from the DB.
5. **Streaming I/O Constraints**:
   - Strictly avoid loading files into RAM. Use `io.Copy(file, req.Body)` for uploads and directly serve the file `http.ServeContent` for downloads.
   - Enforce server-side overall upload limits using `http.MaxBytesReader` to cap payload sizes (e.g., 100MB or 1GB) and prevent disk exhaustion/DOS.
6. **Rate Limiting**:
   - Implement an IP-based rate limiter (e.g., max 5 uploads/min per IP) to prevent malicious spam.

## 4. Admin Dashboard (Phase 2)

- **Endpoint**: `/admin/*` (Protected by strict JSON Web Tokens or external basic HTTP Auth/OIDC proxy).
- **Stats Collection**:
  - Total files active on disk.
  - Total storage payload used.
  - Overall bandwidth consumed (calculated safely via an `io.Writer` byte counting wrapper around the responder stream).
  - Active connections (measured via server middleware).
- **Settings Management**: Access to safely adjust global `Max-Downloads` caps and maximum `Expiry` bounds during runtime.
- **Privacy Design**: 
  - The dashboard displays non-sensitive file metadata ONLY (Size, Upload Time, UUID, Status).
  - Designed completely decoupled from target interactions. The Admin UI must not offer any endpoint or capability to download or inspect the physical file contents.
