package cmd

import (
	"fmt"
	"os"

	"srvx/internal/auth"

	"github.com/spf13/cobra"
)

var (
	rootDir  string
	username string
	password string
	tlsCert  string
	tlsKey   string
)

var rootCmd = &cobra.Command{
	Use:   "fileserver",
	Short: "Multi-protocol file server (HTTP / SFTP / FTP / NFS)",
	Long:  "A unified file server supporting HTTP, SFTP, FTP, and NFS protocols via subcommands.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootDir, "root", "r", ".", "root directory to share")
	rootCmd.PersistentFlags().StringVarP(&username, "user", "u", "", "username for authentication (empty = anonymous)")
	rootCmd.PersistentFlags().StringVarP(&password, "pass", "p", "", "password for authentication")
	rootCmd.PersistentFlags().StringVar(&tlsCert, "tls-cert", "", "path to TLS certificate file")
	rootCmd.PersistentFlags().StringVar(&tlsKey, "tls-key", "", "path to TLS key file")
}

func buildAuthStore() *auth.Store {
	store := auth.NewStore()
	if username != "" {
		store.AddUser(username, password)
	}
	return store
}

func hasTLS() bool {
	return tlsCert != "" && tlsKey != ""
}
