package cmd

import (
	"fmt"

	"srvx/internal/httpfs"

	"github.com/spf13/cobra"
)

var httpsPort int
var httpsUpload bool
var httpsVerbose bool

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
		if httpsVerbose {
			opts = append(opts, httpfs.WithVerbose())
		}
		srv := httpfs.New(rootDir, addr, buildAuthStore(), opts...)
		return srv.ListenAndServe()
	},
}

func init() {
	httpsCmd.Flags().IntVarP(&httpsPort, "port", "P", 8443, "listen port")
	httpsCmd.Flags().BoolVar(&httpsUpload, "upload", false, "enable file upload (PUT) and delete (DELETE)")
	httpsCmd.Flags().BoolVarP(&httpsVerbose, "verbose", "v", false, "log received request headers and body")
	rootCmd.AddCommand(httpsCmd)
}
