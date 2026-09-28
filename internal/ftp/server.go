package ftp

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"srvx/internal/auth"

	ftpserver "github.com/goftp/server"
)

type Server struct {
	root    string
	addr    string
	auth    *auth.Store
	cert    string
	key     string
	certPEM []byte
	keyPEM  []byte
}

type Option func(*Server)

func WithTLS(cert, key string) Option {
	return func(s *Server) { s.cert = cert; s.key = key }
}

func WithTLSBytes(cert, key []byte) Option {
	return func(s *Server) { s.certPEM = cert; s.keyPEM = key }
}

func New(root, addr string, store *auth.Store, opts ...Option) *Server {
	s := &Server{root: root, addr: addr, auth: store}
	for _, o := range opts {
		o(s)
	}
	return s
}

type driverFactory struct {
	root string
}

func (f *driverFactory) NewDriver() (ftpserver.Driver, error) {
	return &driver{root: f.root}, nil
}

type driver struct {
	root string
	conn *ftpserver.Conn
}

func (d *driver) Init(c *ftpserver.Conn) { d.conn = c }

func (d *driver) realPath(p string) string {
	clean := filepath.Clean("/" + strings.TrimPrefix(p, "/"))
	return filepath.Join(d.root, clean)
}

func (d *driver) Stat(p string) (ftpserver.FileInfo, error) {
	full := d.realPath(p)
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	return &fileInfo{FileInfo: info, owner: "ftp", group: "ftp"}, nil
}

func (d *driver) ChangeDir(p string) error {
	full := d.realPath(p)
	info, err := os.Stat(full)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", p)
	}
	return nil
}

func (d *driver) ListDir(p string, cb func(ftpserver.FileInfo) error) error {
	full := d.realPath(p)
	entries, err := os.ReadDir(full)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := cb(&fileInfo{FileInfo: info, owner: "ftp", group: "ftp"}); err != nil {
			return err
		}
	}
	return nil
}

func (d *driver) DeleteDir(p string) error {
	return os.RemoveAll(d.realPath(p))
}

func (d *driver) DeleteFile(p string) error {
	return os.Remove(d.realPath(p))
}

func (d *driver) Rename(from, to string) error {
	return os.Rename(d.realPath(from), d.realPath(to))
}

func (d *driver) MakeDir(p string) error {
	return os.MkdirAll(d.realPath(p), 0o755)
}

func (d *driver) GetFile(p string, offset int64) (int64, io.ReadCloser, error) {
	full := d.realPath(p)
	f, err := os.Open(full)
	if err != nil {
		return 0, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return 0, nil, err
	}
	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			f.Close()
			return 0, nil, err
		}
	}
	return info.Size(), f, nil
}

func (d *driver) PutFile(p string, r io.Reader, appendData bool) (int64, error) {
	full := d.realPath(p)
	mode := os.O_CREATE | os.O_WRONLY
	if appendData {
		mode |= os.O_APPEND
	} else {
		mode |= os.O_TRUNC
	}
	f, err := os.OpenFile(full, mode, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)
}

type fileInfo struct {
	os.FileInfo
	owner string
	group string
}

func (f *fileInfo) Owner() string { return f.owner }
func (f *fileInfo) Group() string { return f.group }

type ftpAuth struct {
	store *auth.Store
}

func (a *ftpAuth) CheckPasswd(user, pass string) (bool, error) {
	if a.store == nil || len(a.store.Users()) == 0 {
		return true, nil
	}
	return a.store.Check(user, pass), nil
}

func (s *Server) resolveTLSFiles() (cert, key string, cleanup func(), err error) {
	cleanup = func() {}
	if s.cert != "" && s.key != "" {
		return s.cert, s.key, cleanup, nil
	}
	if len(s.certPEM) == 0 || len(s.keyPEM) == 0 {
		return "", "", cleanup, nil
	}

	certFile, err := os.CreateTemp("", "srvx-ftps-*.crt")
	if err != nil {
		return "", "", cleanup, err
	}
	if _, err := certFile.Write(s.certPEM); err != nil {
		certFile.Close()
		os.Remove(certFile.Name())
		return "", "", cleanup, err
	}
	certFile.Close()

	keyFile, err := os.CreateTemp("", "srvx-ftps-*.key")
	if err != nil {
		os.Remove(certFile.Name())
		return "", "", cleanup, err
	}
	if _, err := keyFile.Write(s.keyPEM); err != nil {
		keyFile.Close()
		os.Remove(keyFile.Name())
		os.Remove(certFile.Name())
		return "", "", cleanup, err
	}
	keyFile.Close()

	cleanup = func() {
		os.Remove(certFile.Name())
		os.Remove(keyFile.Name())
	}
	return certFile.Name(), keyFile.Name(), cleanup, nil
}

func (s *Server) ListenAndServe() error {
	host, port, err := net.SplitHostPort(s.addr)
	if err != nil {
		return fmt.Errorf("parse address: %w", err)
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("parse port: %w", err)
	}

	certFile, keyFile, cleanup, err := s.resolveTLSFiles()
	if err != nil {
		return fmt.Errorf("prepare TLS files: %w", err)
	}
	defer cleanup()

	opts := &ftpserver.ServerOpts{
		Factory:  &driverFactory{root: s.root},
		Auth:     &ftpAuth{store: s.auth},
		Name:     "Go File Server (FTP)",
		Hostname: host,
		Port:     p,
	}
	if certFile != "" && keyFile != "" {
		opts.TLS = true
		opts.CertFile = certFile
		opts.KeyFile = keyFile
		opts.ExplicitFTPS = true
		log.Printf("FTP server listening on %s (FTPS)", s.addr)
	} else {
		log.Printf("FTP server listening on %s", s.addr)
	}

	srv := ftpserver.NewServer(opts)
	return srv.ListenAndServe()
}
