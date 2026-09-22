package httpfs

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"server/internal/auth"
)

type Server struct {
	root   string
	addr   string
	auth   *auth.Store
	cert   string
	key    string
	upload bool
}

type Option func(*Server)

func WithTLS(cert, key string) Option {
	return func(s *Server) { s.cert = cert; s.key = key }
}

func WithUpload() Option {
	return func(s *Server) { s.upload = true }
}

func New(root, addr string, store *auth.Store, opts ...Option) *Server {
	s := &Server{root: root, addr: addr, auth: store}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Server) basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.auth == nil || len(s.auth.Users()) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || !s.auth.Check(user, pass) {
			w.Header().Set("WWW-Authenticate", `Basic realm="fileserver"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()
	if s.upload {
		mux.HandleFunc("/", s.handleFile)
	} else {
		mux.Handle("/", s.basicAuth(http.FileServer(http.Dir(s.root))))
	}
	return mux
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil && len(s.auth.Users()) > 0 {
		user, pass, ok := r.BasicAuth()
		if !ok || !s.auth.Check(user, pass) {
			w.Header().Set("WWW-Authenticate", `Basic realm="fileserver"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		http.FileServer(http.Dir(s.root)).ServeHTTP(w, r)
	case http.MethodPut:
		s.doUpload(w, r)
	case http.MethodDelete:
		s.doDelete(w, r)
	case http.MethodOptions:
		w.Header().Set("Allow", "GET, HEAD, PUT, DELETE, OPTIONS")
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) safePath(p string) (string, bool) {
	clean := filepath.Clean(filepath.Join(s.root, strings.TrimPrefix(p, "/")))
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", false
	}
	rootAbs, _ := filepath.Abs(s.root)
	if !strings.HasPrefix(abs, rootAbs) {
		return "", false
	}
	return abs, true
}

func (s *Server) doUpload(w http.ResponseWriter, r *http.Request) {
	full, ok := s.safePath(r.URL.Path)
	if !ok {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f, err := os.Create(full)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	if _, err := f.ReadFrom(r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) doDelete(w http.ResponseWriter, r *http.Request) {
	full, ok := s.safePath(r.URL.Path)
	if !ok {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	if err := os.RemoveAll(full); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ListenAndServe() error {
	handler := s.handler()
	srv := &http.Server{Addr: s.addr, Handler: handler}
	if s.cert != "" && s.key != "" {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		log.Printf("HTTP server listening on %s (TLS)", s.addr)
		return srv.ListenAndServeTLS(s.cert, s.key)
	}
	log.Printf("HTTP server listening on %s", s.addr)
	return srv.ListenAndServe()
}
