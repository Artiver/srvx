package cmd

import (
	"fmt"

	"server/internal/httpfs"

	"github.com/spf13/cobra"
)

var httpPort int
var httpUpload bool

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Start HTTP file server",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf("0.0.0.0:%d", httpPort)
		opts := []httpfs.Option{}
		if hasTLS() {
			opts = append(opts, httpfs.WithTLS(tlsCert, tlsKey))
		}
		if httpUpload {
			opts = append(opts, httpfs.WithUpload())
		}
		srv := httpfs.New(rootDir, addr, buildAuthStore(), opts...)
		return srv.ListenAndServe()
	},
}

func init() {
	httpCmd.Flags().IntVarP(&httpPort, "port", "P", 8080, "listen port")
	httpCmd.Flags().BoolVar(&httpUpload, "upload", false, "enable file upload (PUT) and delete (DELETE)")
	rootCmd.AddCommand(httpCmd)
}
