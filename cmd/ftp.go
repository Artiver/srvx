package cmd

import (
	"fmt"

	"srvx/internal/ftp"

	"github.com/spf13/cobra"
)

var ftpPort int

var ftpCmd = &cobra.Command{
	Use:   "ftp",
	Short: "Start FTP file server",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf("0.0.0.0:%d", ftpPort)
		opts := []ftp.Option{}
		if hasTLS() {
			opts = append(opts, ftp.WithTLS(tlsCert, tlsKey))
		}
		srv := ftp.New(rootDir, addr, buildAuthStore(), opts...)
		return srv.ListenAndServe()
	},
}

func init() {
	ftpCmd.Flags().IntVarP(&ftpPort, "port", "P", 2121, "listen port")
	rootCmd.AddCommand(ftpCmd)
}
