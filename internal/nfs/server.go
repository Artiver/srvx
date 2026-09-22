package nfs

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	nfsserver "github.com/willscott/go-nfs"
	nfshelper "github.com/willscott/go-nfs/helpers"
)

type Server struct {
	root string
	addr string
}

func New(root, addr string) *Server {
	return &Server{root: root, addr: addr}
}

type changeFS struct {
	billy.Filesystem
}

func (fs changeFS) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(fs.Join(fs.Root(), name), mode)
}

func (fs changeFS) Lchown(name string, uid, gid int) error {
	return os.Lchown(fs.Join(fs.Root(), name), uid, gid)
}

func (fs changeFS) Chown(name string, uid, gid int) error {
	return os.Chown(fs.Join(fs.Root(), name), uid, gid)
}

func (fs changeFS) Chtimes(name string, atime, mtime time.Time) error {
	return os.Chtimes(fs.Join(fs.Root(), name), atime, mtime)
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	log.Printf("NFS server listening on %s, exporting %s", s.addr, s.root)

	bfs := osfs.New(s.root)
	handler := nfshelper.NewNullAuthHandler(changeFS{bfs})
	cacheHandler := nfshelper.NewCachingHandler(handler, 1024)
	return nfsserver.Serve(listener, cacheHandler)
}
