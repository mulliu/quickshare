package server

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mulliu/quickshare/internal/fstore"
)

//go:embed home.html
var homeTemplate embed.FS

type Server struct {
	store         *fstore.Store
	lanIP         string
	port          int
	maxSize       int64
	tmpl          *template.Template
	qrPNG         []byte
	srv           *http.Server
	logFile       *os.File
	lastHeartbeat time.Time
	heartbeatMu   sync.Mutex
	texts         map[string]textEntry
	textsMu       sync.Mutex
}

type textEntry struct {
	ID        string    `json:"id"`
	Content   string    `json:"-"`
	Preview   string    `json:"preview"`
	CreatedAt time.Time `json:"created_at"`
}

func New(store *fstore.Store, lanIP string, port int, maxSize int64, qrPNG []byte) (*Server, error) {
	tmpl, err := template.ParseFS(homeTemplate, "home.html")
	if err != nil {
		return nil, err
	}

	exeDir, _ := filepath.Split(os.Args[0])
	logPath := filepath.Join(exeDir, "quickshare.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err == nil {
		log.SetOutput(f)
	}

	return &Server{
		store:   store,
		lanIP:   lanIP,
		port:    port,
		maxSize: maxSize,
		tmpl:    tmpl,
		qrPNG:   qrPNG,
		logFile: f,
		texts:   make(map[string]textEntry),
	}, nil
}

func (s *Server) Serve(listener net.Listener) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.Home)
	mux.HandleFunc("GET /qr.png", s.QRImage)
	mux.HandleFunc("POST /upload", s.UploadFile)
	mux.HandleFunc("GET /download/{id}", s.DownloadFile)
	mux.HandleFunc("GET /files", s.ListFiles)
	mux.HandleFunc("POST /share-text", s.ShareText)
	mux.HandleFunc("GET /latest-text", s.LatestText)
	mux.HandleFunc("POST /shutdown", s.ShutdownHandler)
	mux.HandleFunc("GET /heartbeat", s.Heartbeat)

	s.lastHeartbeat = time.Now()
	go s.watchdog()

	addr := fmt.Sprintf("%s:%d", s.lanIP, s.port)
	s.srv = &http.Server{
		Handler:      withLogging(corsMiddleware(mux)),
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  0,
	}

	log.Printf("Listening on %s", addr)
	return s.srv.Serve(listener)
}

func (s *Server) Close() error {
	if s.logFile != nil {
		s.logFile.Close()
	}
	if s.srv != nil {
		return s.srv.Close()
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) ShutdownHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Server shutting down..."))
	go s.srv.Shutdown(context.Background())
}

func (s *Server) Heartbeat(w http.ResponseWriter, r *http.Request) {
	s.heartbeatMu.Lock()
	s.lastHeartbeat = time.Now()
	s.heartbeatMu.Unlock()
}

func (s *Server) watchdog() {
	const heartbeatTimeout = 8 * time.Second

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.heartbeatMu.Lock()
		stale := time.Since(s.lastHeartbeat) > heartbeatTimeout
		s.heartbeatMu.Unlock()
		if stale {
			log.Printf("No clients detected for %s, shutting down", heartbeatTimeout)
			s.srv.Shutdown(context.Background())
			return
		}
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingRW{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(lrw, r)
		log.Printf("%s %s %d %s [%s]", r.Method, r.URL.Path, lrw.code, time.Since(start).Round(time.Millisecond), r.RemoteAddr)
	})
}

type loggingRW struct {
	http.ResponseWriter
	code int
}

func (w *loggingRW) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
