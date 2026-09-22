package cmd

import (
	"fmt"

	"server/internal/sftp"

	"github.com/spf13/cobra"
)

var sftpPort int
var sftpHostKey string

var sftpCmd = &cobra.Command{
	Use:   "sftp",
	Short: "Start SFTP file server",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf("0.0.0.0:%d", sftpPort)
		opts := []sftp.Option{}
		if sftpHostKey != "" {
			opts = append(opts, sftp.WithHostKey(sftpHostKey))
		}
		if hasTLS() {
			opts = append(opts, sftp.WithTLS(tlsCert, tlsKey))
		}
		srv := sftp.New(rootDir, addr, buildAuthStore(), opts...)
		return srv.ListenAndServe()
	},
}

func init() {
	sftpCmd.Flags().IntVarP(&sftpPort, "port", "P", 2022, "listen port")
	sftpCmd.Flags().StringVar(&sftpHostKey, "host-key", "", "path to SSH host private key (empty = auto-generate ephemeral key)")
	rootCmd.AddCommand(sftpCmd)
}
