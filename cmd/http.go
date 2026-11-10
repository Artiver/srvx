package cmd

import (
	"fmt"

	"srvx/internal/httpfs"

	"github.com/spf13/cobra"
)

var httpPort int
var httpUpload bool
var httpVerbose bool

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Start HTTP file server",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf("0.0.0.0:%d", httpPort)
		opts := []httpfs.Option{}
		if httpUpload {
			opts = append(opts, httpfs.WithUpload())
		}
		if httpVerbose {
			opts = append(opts, httpfs.WithVerbose())
		}
		srv := httpfs.New(rootDir, addr, buildAuthStore(), opts...)
		return srv.ListenAndServe()
	},
}

func init() {
	httpCmd.Flags().IntVarP(&httpPort, "port", "P", 8080, "listen port")
	httpCmd.Flags().BoolVar(&httpUpload, "upload", false, "enable file upload (PUT) and delete (DELETE)")
	httpCmd.Flags().BoolVarP(&httpVerbose, "verbose", "v", false, "log received request headers and body")
	rootCmd.AddCommand(httpCmd)
}
