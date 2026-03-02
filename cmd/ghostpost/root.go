// cmd/ghostpost/root.go

package main

import (
	"os"

	"github.com/rodchristiansen/ghost-gitops-publishing/internal/config"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=YYYY.MM.DD.HHMM"
var version = "dev"

var cfg *config.Config

func main() {
	root := &cobra.Command{
		Use:     "ghostpost",
		Short:   "Git-first publishing to Ghost",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			cfg, err = config.Load(cmd)
			return err
		},
	}

	root.PersistentFlags().String("api-url", "", "Ghost Admin API base URL (https://blog.example/ghost/api/admin/)")
	root.PersistentFlags().String("admin-jwt", "", "Admin API JWT")

	root.AddCommand(publishCmd())
	root.AddCommand(tagsCmd())
	root.AddCommand(imagesCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
