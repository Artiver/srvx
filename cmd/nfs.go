package cmd

import (
	"fmt"

	"server/internal/nfs"

	"github.com/spf13/cobra"
)

var nfsPort int

var nfsCmd = &cobra.Command{
	Use:   "nfs",
	Short: "Start NFS v3 file server",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := fmt.Sprintf("0.0.0.0:%d", nfsPort)
		srv := nfs.New(rootDir, addr)
		return srv.ListenAndServe()
	},
}

func init() {
	nfsCmd.Flags().IntVarP(&nfsPort, "port", "P", 2049, "listen port")
	rootCmd.AddCommand(nfsCmd)
}
