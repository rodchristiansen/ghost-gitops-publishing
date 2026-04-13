// internal/config/loader.go

package config

import (
	"path/filepath"
	"strings"

	"github.com/rodchristiansen/ghost-gitops-publishing/internal/auth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Load(cmd *cobra.Command) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config") // config.{json,yaml,toml}
	v.AddConfigPath("$HOME/.ghostpost")
	v.AddConfigPath(".")
	v.SetEnvPrefix("ghost")
	v.AutomaticEnv()

	v.SetDefault("api_version", "v5")

	_ = v.BindPFlag("api_url", cmd.Flags().Lookup("api-url"))
	_ = v.BindPFlag("admin_jwt", cmd.Flags().Lookup("admin-jwt"))
	_ = v.BindPFlag("api_version", cmd.Flags().Lookup("api-version"))

	_ = v.ReadInConfig() // ignore “file not found”

	cfg := &Config{
		APIURL:     v.GetString("api_url"),
		AdminJWT:   v.GetString("admin_jwt"),
		APIVersion: v.GetString("api_version"),
	}

	// Accept raw Admin API key and auto-sign it.
	if strings.Contains(cfg.AdminJWT, ":") {
		if signed, err := auth.FromKey(cfg.AdminJWT, cfg.APIURL, cfg.APIVersion); err == nil && signed != "" {
			cfg.AdminJWT = signed
		}
	}

	// Ensure trailing slash on API URL
	cfg.APIURL = filepath.ToSlash(cfg.APIURL) + "/"

	return cfg, nil
}
