package cmd

import (
	"fmt"

	"srvx/internal/sftp"

	"github.com/spf13/cobra"
)

var sftpPort int
var sftpHostKey string
var sftpHostKeyType string
var sftpBanner string

var sftpCmd = &cobra.Command{
	Use:   "sftp",
	Short: "Start SFTP file server",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf("0.0.0.0:%d", sftpPort)
		opts := []sftp.Option{}
		if sftpHostKey != "" {
			opts = append(opts, sftp.WithHostKey(sftpHostKey))
		} else {
			switch sftpHostKeyType {
			case "rsa":
				if len(defaultRSAHostKey) > 0 {
					opts = append(opts, sftp.WithHostKeyBytes(defaultRSAHostKey))
				}
			default:
				if len(defaultHostKey) > 0 {
					opts = append(opts, sftp.WithHostKeyBytes(defaultHostKey))
				}
			}
		}
		if sftpBanner != "" {
			opts = append(opts, sftp.WithBanner(sftpBanner))
		}
		srv := sftp.New(rootDir, addr, buildAuthStore(), opts...)
		return srv.ListenAndServe()
	},
}

func init() {
	sftpCmd.Flags().IntVarP(&sftpPort, "port", "P", 2022, "listen port")
	sftpCmd.Flags().StringVar(&sftpHostKey, "key", "", "path to SSH host private key (empty = use embedded default key)")
	sftpCmd.Flags().StringVar(&sftpHostKeyType, "type", "ed25519", "embedded default host key type: ed25519 or rsa")
	sftpCmd.Flags().StringVar(&sftpBanner, "banner", "", "SSH banner sent to clients before authentication")
	rootCmd.AddCommand(sftpCmd)
}
