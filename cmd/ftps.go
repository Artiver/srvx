package cmd

import (
	"fmt"

	"srvx/internal/ftp"

	"github.com/spf13/cobra"
)

var ftpsPort int

var ftpsCmd = &cobra.Command{
	Use:   "ftps",
	Short: "Start FTPS file server",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf("0.0.0.0:%d", ftpsPort)
		opts := []ftp.Option{}
		if hasTLS() {
			opts = append(opts, ftp.WithTLS(tlsCert, tlsKey))
		} else if hasDefaultTLS() {
			opts = append(opts, ftp.WithTLSBytes(defaultCert, defaultKey))
		}
		srv := ftp.New(rootDir, addr, buildAuthStore(), opts...)
		return srv.ListenAndServe()
	},
}

func init() {
	ftpsCmd.Flags().IntVarP(&ftpsPort, "port", "P", 990, "listen port")
	rootCmd.AddCommand(ftpsCmd)
}
