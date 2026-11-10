package httpfs

import (
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"srvx/internal/auth"
)

const verboseBodyLimit = 100

type Server struct {
	root     string
	addr     string
	auth     *auth.Store
	certPath string
	keyPath  string
	certPEM  []byte
	keyPEM   []byte
	upload   bool
	verbose  bool
}

type Option func(*Server)

func WithTLS(cert, key string) Option {
	return func(s *Server) { s.certPath = cert; s.keyPath = key }
}

func WithTLSBytes(cert, key []byte) Option {
	return func(s *Server) { s.certPEM = cert; s.keyPEM = key }
}

func WithUpload() Option {
	return func(s *Server) { s.upload = true }
}

func WithVerbose() Option {
	return func(s *Server) { s.verbose = true }
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
	if s.verbose {
		return s.verboseLog(mux)
	}
	return mux
}

type readCloser struct {
	io.Reader
	io.Closer
}

func (s *Server) verboseLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		ip, port, _ := net.SplitHostPort(r.RemoteAddr)
		fmt.Fprintf(&b, "Request from [%s]:%s\n", ip, port)
		fmt.Fprintf(&b, "%s %s %s\n", r.Method, r.URL.RequestURI(), r.Proto)
		fmt.Fprintf(&b, "Host: %s\n", r.Host)
		keys := make([]string, 0, len(r.Header))
		for k := range r.Header {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, v := range r.Header.Values(k) {
				fmt.Fprintf(&b, "%s: %s\n", k, v)
			}
		}
		if r.Body != nil && r.Body != http.NoBody {
			ct := r.Header.Get("Content-Type")
			if isFileContentType(ct) {
				prefix := make([]byte, verboseBodyLimit)
				n, err := io.ReadFull(r.Body, prefix)
				if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
					log.Printf("read request body: %v", err)
				}
				prefix = prefix[:n]
				r.Body = &readCloser{Reader: io.MultiReader(bytes.NewReader(prefix), r.Body), Closer: r.Body}
				fmt.Fprintf(&b, "body (%s, hex, first %d bytes):\n%s", ct, n, hex.Dump(prefix))
			} else {
				data, err := io.ReadAll(r.Body)
				if err != nil {
					log.Printf("read request body: %v", err)
				}
				r.Body = io.NopCloser(bytes.NewReader(data))
				fmt.Fprintf(&b, "\n%s\n", data)
			}
		}
		log.Print(b.String())
		next.ServeHTTP(w, r)
	})
}

func isFileContentType(ct string) bool {
	if ct == "" {
		return false
	}
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		mt = strings.TrimSpace(strings.SplitN(ct, ";", 2)[0])
	}
	mt = strings.ToLower(mt)
	switch {
	case strings.HasPrefix(mt, "text/"):
		return false
	case mt == "application/json",
		mt == "application/xml",
		mt == "application/xhtml+xml",
		mt == "application/javascript",
		mt == "application/x-javascript",
		mt == "application/x-www-form-urlencoded",
		mt == "application/yaml",
		mt == "application/x-yaml":
		return false
	}
	return true
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

func (s *Server) tlsCertificate() (tls.Certificate, bool, error) {
	if len(s.certPEM) > 0 && len(s.keyPEM) > 0 {
		c, err := tls.X509KeyPair(s.certPEM, s.keyPEM)
		return c, true, err
	}
	if s.certPath != "" && s.keyPath != "" {
		c, err := tls.LoadX509KeyPair(s.certPath, s.keyPath)
		return c, true, err
	}
	return tls.Certificate{}, false, nil
}

func (s *Server) ListenAndServe() error {
	handler := s.handler()
	srv := &http.Server{Addr: s.addr, Handler: handler}
	cert, hasCert, err := s.tlsCertificate()
	if err != nil {
		return fmt.Errorf("load TLS certificate: %w", err)
	}
	if hasCert {
		srv.TLSConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
		log.Printf("HTTPS server listening on %s (TLS)", s.addr)
		return srv.ListenAndServeTLS("", "")
	}
	log.Printf("HTTP server listening on %s", s.addr)
	return srv.ListenAndServe()
}
