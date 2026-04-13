// cmd/ghostpost/images.go

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rodchristiansen/ghost-gitops-publishing/internal/images"
	"github.com/spf13/cobra"
)

func imagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Image operations",
	}
	cmd.AddCommand(imagesUploadCmd())
	return cmd
}

func imagesUploadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upload <file> [file...]",
		Short: "Upload images to Ghost and print their URLs",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			svc := images.New(cfg.APIURL, cfg.AdminJWT, httpClient)

			// Expand globs so users can pass e.g. "ready/*.png"
			var paths []string
			for _, arg := range args {
				matches, err := filepath.Glob(arg)
				if err != nil || len(matches) == 0 {
					paths = append(paths, arg) // let Upload surface the real error
				} else {
					paths = append(paths, matches...)
				}
			}

			exitCode := 0
			for _, p := range paths {
				abs, err := filepath.Abs(p)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %s: %v\n", p, err)
					exitCode = 1
					continue
				}
				url, err := svc.Upload(abs)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error: %s: %v\n", p, err)
					exitCode = 1
					continue
				}
				fmt.Printf("%s\t%s\n", p, url)
			}
			if exitCode != 0 {
				os.Exit(exitCode)
			}
			return nil
		},
	}
}
