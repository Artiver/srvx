package sftp

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"srvx/internal/auth"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	root       string
	addr       string
	auth       *auth.Store
	hostKey    string
	hostKeyPEM []byte
}

type Option func(*Server)

func WithHostKey(path string) Option {
	return func(s *Server) { s.hostKey = path }
}

func WithHostKeyBytes(pem []byte) Option {
	return func(s *Server) { s.hostKeyPEM = pem }
}

func New(root, addr string, store *auth.Store, opts ...Option) *Server {
	s := &Server{root: root, addr: addr, auth: store}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Server) loadHostKey() (ssh.Signer, error) {
	var data []byte
	if s.hostKey != "" {
		b, err := os.ReadFile(s.hostKey)
		if err != nil {
			return nil, fmt.Errorf("read host key: %w", err)
		}
		data = b
	} else if len(s.hostKeyPEM) > 0 {
		data = s.hostKeyPEM
	} else {
		return nil, fmt.Errorf("no host key provided")
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse host key: %w", err)
	}
	return signer, nil
}

func (s *Server) sshConfig() (*ssh.ServerConfig, error) {
	signer, err := s.loadHostKey()
	if err != nil {
		return nil, err
	}
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if s.auth == nil || len(s.auth.Users()) == 0 {
				return nil, nil
			}
			if s.auth.Check(c.User(), string(pass)) {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}
	config.AddHostKey(signer)
	return config, nil
}

func (s *Server) ListenAndServe() error {
	config, err := s.sshConfig()
	if err != nil {
		return err
	}

	var listener net.Listener
	listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	log.Printf("SFTP server listening on %s", s.addr)
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn, config)
	}
}

func (s *Server) handleConn(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()
	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.User())
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}
		go s.handleSession(channel, requests)
	}
}

func (s *Server) handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()
	for req := range requests {
		ok := false
		switch req.Type {
		case "subsystem":
			if string(req.Payload[4:]) == "sftp" {
				ok = true
				go s.serveSFTP(channel)
			}
		}
		req.Reply(ok, nil)
	}
}

func (s *Server) serveSFTP(channel ssh.Channel) {
	root, err := filepath.Abs(s.root)
	if err != nil {
		log.Printf("resolve root: %v", err)
		return
	}
	handler, err := sftp.NewServer(channel, sftp.WithServerWorkingDirectory(root))
	if err != nil {
		log.Printf("create sftp server: %v", err)
		return
	}
	if err := handler.Serve(); err != nil {
		log.Printf("SFTP serve error: %v", err)
	}
	handler.Close()
}
