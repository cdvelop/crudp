# File Upload Handling: "Upload & Reference" Pattern

**Integration with Hybrid Handler Registration System**

This guide demonstrates how to implement file uploads within CRUDP's hybrid handler registration system, using `HttpRouteProvider` for HTTP routes while keeping CRUD handlers decoupled from transport details.

---

## **Core Pattern: "Upload & Reference"**

**Why not pass `http.ResponseWriter` to handlers?**

Passing `w` to handlers breaks the asynchronous architecture and clean separation:

1. **Async Processing Break:** HTTP connections close before background workers finish
2. **Transport Coupling:** Handlers become HTTP-only, untestable without mocks
3. **Single Responsibility:** HTTP layer handles network I/O, CRUD handlers manage business logic

**Solution:** HTTP routes handle physical storage, CRUD handlers receive file references.

---

## **1. File Reference Structure**

**File: `modules/files/files.go`**
```go
package files

import (
    "context"
)

// FileReference represents uploaded file metadata (not the file itself)
type FileReference struct {
    ID        string // Unique file identifier
    Path      string // Storage path (/uploads/uuid.jpg)
    Name      string // Original filename
    Size      int64  // File size in bytes
    MimeType  string // MIME type (image/jpeg, etc.)
    UploadedAt string // ISO timestamp
}

type Handler struct{}

// Implement CRUDP binary protocol interfaces for file metadata management
func (h *Handler) Create(ctx context.Context, data ...any) (any, error) {
    ref := data[0].(FileReference)
    // Business logic: save file reference to database
    return h.db.SaveFileReference(ref)
}

func (h *Handler) Read(ctx context.Context, data ...any) (any, error) {
    // Query file references from database
    return h.db.GetFileReferences()
}

func (h *Handler) Update(ctx context.Context, data ...any) (any, error) {
    // Update file metadata (rename, move, etc.)
    ref := data[0].(FileReference)
    return h.db.UpdateFileReference(ref)
}

func (h *Handler) Delete(ctx context.Context, data ...any) (any, error) {
    // Mark file for deletion or move to trash
    fileID := data[0].(string)
    return h.db.DeleteFileReference(fileID)
}
```

---

## **2. HTTP Route Implementation**

**File: `modules/files/back.server.go`**
```go
//go:build !wasm

package files

import (
    "crypto/rand"
    "encoding/hex"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// Implement HttpRouteProvider interface
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
    mux.Handle("POST /files/upload", http.HandlerFunc(h.handleFileUpload))
    mux.Handle("GET /files/download/", http.HandlerFunc(h.handleFileDownload))
    mux.Handle("DELETE /files/delete/", http.HandlerFunc(h.handleFileDelete))
}

// handleFileUpload processes multipart form uploads
func (h *Handler) handleFileUpload(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Parse multipart form (max 32MB)
    err := r.ParseMultipartForm(32 << 20)
    if err != nil {
        http.Error(w, "Failed to parse form", http.StatusBadRequest)
        return
    }

    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "No file provided", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Validate file size (example: max 10MB)
    if header.Size > 10<<20 {
        http.Error(w, "File too large", http.StatusBadRequest)
        return
    }

    // Generate unique filename
    id := generateID()
    ext := filepath.Ext(header.Filename)
    filename := id + ext
    storagePath := filepath.Join("/uploads", filename)

    // Ensure upload directory exists
    os.MkdirAll("/uploads", 0755)

    // Save file to disk
    dst, err := os.Create(storagePath)
    if err != nil {
        http.Error(w, "Failed to save file", http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    _, err = io.Copy(dst, file)
    if err != nil {
        http.Error(w, "Failed to write file", http.StatusInternalServerError)
        return
    }

    // Create file reference struct
    fileRef := FileReference{
        ID:        id,
        Path:      storagePath,
        Name:      header.Filename,
        Size:      header.Size,
        MimeType:  header.Header.Get("Content-Type"),
        UploadedAt: time.Now().Format(time.RFC3339),
    }

    // Call CRUD handler with reference (not the file stream)
    // This allows async processing and keeps handlers decoupled from HTTP
    result, err := h.Create(r.Context(), fileRef)
    if err != nil {
        // Cleanup file if database save failed
        os.Remove(storagePath)
        http.Error(w, "Failed to process file", http.StatusInternalServerError)
        return
    }

    // Return success response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"status":"uploaded","file_id":"` + id + `"}`))
}

// handleFileDownload serves uploaded files
func (h *Handler) handleFileDownload(w http.ResponseWriter, r *http.Request) {
    // Extract file ID from URL path
    fileID := strings.TrimPrefix(r.URL.Path, "/files/download/")
    if fileID == "" {
        http.Error(w, "File ID required", http.StatusBadRequest)
        return
    }

    // Get file reference from database
    refs, err := h.Read(r.Context())
    if err != nil {
        http.Error(w, "File not found", http.StatusNotFound)
        return
    }

    // Find matching file
    var fileRef FileReference
    for _, ref := range refs.([]FileReference) {
        if ref.ID == fileID {
            fileRef = ref
            break
        }
    }

    if fileRef.ID == "" {
        http.Error(w, "File not found", http.StatusNotFound)
        return
    }

    // Serve file with original filename
    w.Header().Set("Content-Disposition", `attachment; filename="`+fileRef.Name+`"`)
    http.ServeFile(w, r, fileRef.Path)
}

// handleFileDelete marks file for deletion
func (h *Handler) handleFileDelete(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    fileID := strings.TrimPrefix(r.URL.Path, "/files/delete/")
    if fileID == "" {
        http.Error(w, "File ID required", http.StatusBadRequest)
        return
    }

    // Call CRUD handler to mark for deletion
    _, err := h.Delete(r.Context(), fileID)
    if err != nil {
        http.Error(w, "Failed to delete file", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"deleted"}`))
}

// generateID creates a random hex string for file identification
func generateID() string {
    bytes := make([]byte, 16)
    rand.Read(bytes)
    return hex.EncodeToString(bytes)
}
```

---

## **3. Integration with Handler Registration**

**File: `web/server.go`**
```go
//go:build !wasm

package main

import (
    "net/http"
    "myproject/pkg/router"
)

func main() {
    // Get the complete HTTP handler with all routes and middleware
    // CRUDP setup with file handlers is centralized in pkg/router/router.go
    handler := router.NewRouter()
    
    http.ListenAndServe(":8080", handler)
}
```

---

## **Key Benefits**

- **🔄 Async Processing:** Files can be processed in background queues
- **🧪 Testability:** CRUD handlers tested with mock references, no HTTP
- **🔌 Reusability:** Same logic for HTTP, WebSocket, or local file imports
- **📏 Scalability:** Large files don't consume handler memory
- **🛡️ Security:** Global middleware protects upload endpoints
- **🎯 Separation:** HTTP handles I/O, CRUD manages business logic

---

## **Advanced Patterns**

### **Background Processing**
For large files or processing-intensive tasks, use goroutines:

```go
func (h *Handler) handleFileUpload(w http.ResponseWriter, r *http.Request) {
    // ... file saving code ...

    // Quick response
    w.WriteHeader(http.StatusAccepted)
    w.Write([]byte(`{"status":"processing","file_id":"` + id + `"}`))

    // Background processing
    go func() {
        // Image resizing, virus scanning, etc.
        h.processFileAsync(fileRef)
    }()
}
```

### **Storage Abstractions**
Replace local filesystem with S3, cloud storage, etc.:

```go
type Storage interface {
    Save(file io.Reader, path string) error
    Get(path string) (io.ReadCloser, error)
    Delete(path string) error
}

func (h *Handler) setStorage(s Storage) {
    h.storage = s
}
```

### **Validation & Security**
Add comprehensive validation:

```go
func (h *Handler) validateFile(header *multipart.FileHeader) error {
    // Check MIME type
    allowedTypes := []string{"image/jpeg", "image/png", "application/pdf"}
    if !contains(allowedTypes, header.Header.Get("Content-Type")) {
        return errors.New("invalid file type")
    }

    // Check file size
    if header.Size > h.maxFileSize {
        return errors.New("file too large")
    }

    // Virus scanning, etc.
    return nil
}
```

This pattern ensures clean architecture while providing full file upload capabilities within CRUDP's hybrid system.