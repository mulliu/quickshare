package fstore

import (
	"crypto/rand"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileInfo holds metadata about a shared file.
type FileInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	MimeType  string    `json:"mime_type"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages shared files: in-memory registry + on-disk storage.
type Store struct {
	mu        sync.RWMutex
	files     map[string]*entry
	outputDir string
	ttl       time.Duration
	baseURL   string
	done      chan struct{}
}

type entry struct {
	info FileInfo
	path string // full disk path
}

// New creates a Store. It creates the output directory if needed and starts the TTL cleanup goroutine.
// Call Close to stop cleanup.
func New(outputDir string, ttl time.Duration, baseURL string) (*Store, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}
	s := &Store{
		files:     make(map[string]*entry),
		outputDir: outputDir,
		ttl:       ttl,
		baseURL:   baseURL,
		done:      make(chan struct{}),
	}
	if ttl > 0 {
		go s.cleanupLoop()
	}
	s.ScanExisting()
	return s, nil
}

// Add saves an uploaded file to disk and registers it in the store.
// The reader is streamed directly to disk (no memory buffering).
func (s *Store) Add(originalName string, mimeType string, src io.Reader) (*FileInfo, error) {
	id, err := GenerateID()
	if err != nil {
		return nil, err
	}

	sanitized := sanitizeName(originalName)
	savePath := filepath.Join(s.outputDir, filepath.Base(sanitized))

	// If file exists, prepend ID to avoid collision
	dir := filepath.Dir(savePath)
	savePath = filepath.Join(dir, sanitized)
	if _, err := os.Stat(savePath); err == nil {
		savePath = filepath.Join(dir, id+"_"+sanitized)
	}

	file, err := os.Create(savePath)
	if err != nil {
		return nil, err
	}

	written, err := io.Copy(file, src)
	if err != nil {
		file.Close()
		os.Remove(savePath)
		return nil, err
	}
	file.Close()

	mime := mimeType
	if mime == "" {
		mime = detectMime(sanitized)
	}

	url := s.baseURL + "/download/" + id

	info := FileInfo{
		ID:        id,
		Name:      sanitized,
		Size:      written,
		MimeType:  mime,
		URL:       url,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.files[id] = &entry{info: info, path: savePath}
	s.mu.Unlock()

	return &info, nil
}

// Get retrieves file info and disk path by ID. Returns nil if not found.
func (s *Store) Get(id string) (*FileInfo, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.files[id]
	if !ok {
		return nil, "", false
	}
	return &e.info, e.path, true
}

// List returns all files sorted by creation time (newest first).
// Stale entries (missing on disk) are skipped.
func (s *Store) List() []FileInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, e := range s.files {
		if _, err := os.Stat(e.path); os.IsNotExist(err) {
			delete(s.files, id)
		}
	}
	result := make([]FileInfo, 0, len(s.files))
	for _, e := range s.files {
		result = append(result, e.info)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

// Remove deletes a file from disk and the registry.
func (s *Store) Remove(id string) {
	s.mu.Lock()
	e, ok := s.files[id]
	if ok {
		delete(s.files, id)
	}
	s.mu.Unlock()
	if ok {
		os.Remove(e.path)
	}
}

// ScanExisting registers files already in the output directory.
// Call once at startup to make pre-existing files available for download.
func (s *Store) ScanExisting() {
	entries, err := os.ReadDir(s.outputDir)
	if err != nil {
		return
	}
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") {
			continue
		}
		fi, err := de.Info()
		if err != nil || fi.Size() == 0 {
			continue
		}
		id, err := GenerateID()
		if err != nil {
			continue
		}
		s.mu.Lock()
		s.files[id] = &entry{
			info: FileInfo{
				ID:        id,
				Name:      name,
				Size:      fi.Size(),
				MimeType:  detectMime(name),
				URL:       s.baseURL + "/download/" + id,
				CreatedAt: fi.ModTime(),
			},
			path: filepath.Join(s.outputDir, name),
		}
		s.mu.Unlock()
	}
}

// Close stops the cleanup goroutine.
func (s *Store) Close() {
	close(s.done)
}

// cleanupLoop runs periodically to remove expired files.
func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.done:
			return
		}
	}
}

func (s *Store) cleanup() {
	deadline := time.Now().Add(-s.ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, e := range s.files {
		if e.info.CreatedAt.Before(deadline) {
			os.Remove(e.path)
			delete(s.files, id)
		}
	}
}

// GenerateID creates an 8-character random string (base62, ~295M combinations).
func GenerateID() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

// sanitizeName removes path separators and other risky characters from filenames.
func sanitizeName(name string) string {
	if name == "" {
		return "unnamed"
	}
	cleaned := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		if c == '/' || c == '\\' || c == ':' || c == '*' || c == '?' || c == '"' || c == '<' || c == '>' || c == '|' {
			continue
		}
		cleaned = append(cleaned, c)
	}
	result := string(cleaned)
	if result == "" {
		return "unnamed"
	}
	return result
}

// detectMime returns a MIME type based on file extension.
func detectMime(name string) string {
	ext := ""
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			ext = name[i:]
			break
		}
	}
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".gz":
		return "application/gzip"
	case ".tar":
		return "application/x-tar"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".doc", ".docx":
		return "application/msword"
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
