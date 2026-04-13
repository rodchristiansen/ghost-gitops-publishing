// internal/config/config.go

package config

type Config struct {
	APIURL     string
	AdminJWT   string
	APIVersion string // "v5", "v6", etc. — segment Ghost checks in the JWT aud claim
}
