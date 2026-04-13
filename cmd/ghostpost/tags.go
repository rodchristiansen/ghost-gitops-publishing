// cmd/ghostpost/tags.go

package main

import (
	"context"
	"fmt"

	"github.com/rodchristiansen/ghost-gitops-publishing/internal/api"
	"github.com/spf13/cobra"
)

func tagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "Tag operations",
	}
	cmd.AddCommand(tagsListCmd())
	return cmd
}

func tagsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tags on the Ghost site",
		RunE: func(_ *cobra.Command, _ []string) error {
			client := api.New(cfg.APIURL, cfg.AdminJWT)
			tags, err := client.ListTags(context.Background())
			if err != nil {
				return err
			}
			for _, t := range tags {
				fmt.Printf("%s\t%s\n", t.Slug, t.Name)
			}
			return nil
		},
	}
}
