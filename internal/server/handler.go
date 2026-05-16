package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/user/quickshare/internal/fstore"
)

func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	baseURL := fmt.Sprintf("http://%s:%d", s.lanIP, s.port)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := s.tmpl.Execute(w, map[string]string{
		"BaseURL": baseURL,
		"LANIP":   s.lanIP,
		"Port":    fmt.Sprintf("%d", s.port),
	})
	if err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) QRImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(s.qrPNG)
}

func (s *Server) UploadFile(w http.ResponseWriter, r *http.Request) {
	log.Printf("Upload request from %s", r.RemoteAddr)

	r.Body = http.MaxBytesReader(w, r.Body, s.maxSize)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		log.Printf("Upload parse error from %s: %v", r.RemoteAddr, err)
		http.Error(w, fmt.Sprintf("upload error: %v", err), http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Upload no file from %s: %v", r.RemoteAddr, err)
		http.Error(w, fmt.Sprintf("no file in request: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("Upload receiving: %s (%d bytes)", header.Filename, header.Size)

	info, err := s.store.Add(header.Filename, header.Header.Get("Content-Type"), file)
	if err != nil {
		log.Printf("Upload save failed from %s: %v", r.RemoteAddr, err)
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}

	log.Printf("Upload complete: %s (%s) from %s", info.Name, formatBytes(info.Size), r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      info.ID,
		"name":    info.Name,
		"size":    info.Size,
		"url":     info.URL,
	})
}

func (s *Server) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing file id", http.StatusBadRequest)
		return
	}

	info, path, ok := s.store.Get(id)
	if !ok {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "file has been removed from disk", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, info.Name))
	w.Header().Set("Content-Type", info.MimeType)
	http.ServeFile(w, r, path)
}

func (s *Server) ListFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.store.List())
}

func (s *Server) LatestText(w http.ResponseWriter, r *http.Request) {
	s.textsMu.Lock()
	defer s.textsMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if len(s.texts) == 0 {
		json.NewEncoder(w).Encode(nil)
		return
	}
	// Find the most recent text entry
	var latest textEntry
	var latestTime time.Time
	for _, t := range s.texts {
		if t.CreatedAt.After(latestTime) {
			latest = t
			latestTime = t.CreatedAt
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      latest.ID,
		"content": latest.Content,
	})
}

func (s *Server) ShareText(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	content := strings.TrimSpace(req.Text)
	if content == "" {
		http.Error(w, "text is empty", http.StatusBadRequest)
		return
	}

	id, err := fstore.GenerateID()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	preview := content
	if len(preview) > 80 {
		preview = preview[:80] + "…"
	}

	s.textsMu.Lock()
	s.texts[id] = textEntry{
		ID:        id,
		Content:   content,
		Preview:   preview,
		CreatedAt: time.Now(),
	}
	s.textsMu.Unlock()

	log.Printf("ShareText: id=%s (%d chars)", id, len(content))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id,
		"preview": preview,
	})
}

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}
