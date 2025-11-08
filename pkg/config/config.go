package config

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     string // Base URLs like http://pdns1:8081/api/v1  (with /api/v1 at the end)
	APIKey     string // X-API-Key for each server (aligns 1:1 with Servers) OR single entry for all
	ServerID   string // PDNS server-id, usually "localhost"
	AuthToken  string // Optional: wrapper's own Bearer token for clients
	HTTPClient *http.Client
	Addr       string
}

func NewConfig() (*Config, error) {
	serverEnv := mustGetEnv("PDNS_SERVER", "")
	if serverEnv == "" {
		return nil, errors.New("PDNS_SERVER is required . Example: http://pdns1:8081/api/v1")
	}

	cfg := &Config{
		Server:    serverEnv,
		APIKey:    mustGetEnv("PDNS_APIKEY", ""),
		ServerID:  mustGetEnv("PDNS_SERVER_ID", "localhost"),
		AuthToken: mustGetEnv("AUTH_TOKEN", ""), // optional
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		Addr: mustGetEnv("ADDR", ":8080"),
	}

	return cfg, nil
}

func mustGetEnv(key string, def string) string {

	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("No .env file at %s or failed to load: %v\n", ".env", err)
	}
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func (c *Config) PDNSURL(serverBase, path string) string {
	return fmt.Sprintf("%s/servers/%s%s", serverBase, c.ServerID, path)
}
