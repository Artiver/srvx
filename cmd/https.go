package cmd

import (
	"fmt"

	"srvx/internal/httpfs"

	"github.com/spf13/cobra"
)

var httpsPort int
var httpsUpload bool

var httpsCmd = &cobra.Command{
	Use:   "https",
	Short: "Start HTTPS file server",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf("0.0.0.0:%d", httpsPort)
		opts := []httpfs.Option{}
		if hasTLS() {
			opts = append(opts, httpfs.WithTLS(tlsCert, tlsKey))
		} else if hasDefaultTLS() {
			opts = append(opts, httpfs.WithTLSBytes(defaultCert, defaultKey))
		}
		if httpsUpload {
			opts = append(opts, httpfs.WithUpload())
		}
		srv := httpfs.New(rootDir, addr, buildAuthStore(), opts...)
		return srv.ListenAndServe()
	},
}

func init() {
	httpsCmd.Flags().IntVarP(&httpsPort, "port", "P", 8443, "listen port")
	httpsCmd.Flags().BoolVar(&httpsUpload, "upload", false, "enable file upload (PUT) and delete (DELETE)")
	rootCmd.AddCommand(httpsCmd)
}
