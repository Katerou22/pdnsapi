package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Katerou22/pdnsapi/pkg/config"
	"github.com/gofiber/fiber/v2"
)

type Server struct {
	c   *config.Config
	App *fiber.App
}

func NewServer(c *config.Config) *Server {

	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	app.Use(authMiddleware(c.AuthToken))

	return &Server{c: c, App: app}
}

func (s *Server) Run() {

	cfg := s.c
	log.Printf("pdns wrapper listening on %s (server=%v, server-id=%s)", cfg.Addr, cfg.Server, cfg.ServerID)
	if err := s.App.Listen(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
func (s *Server) DoJSON(method, url string, apiKey string, in any, out any) (int, []byte, error) {
	var bodyReader *strings.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = strings.NewReader(string(b))
	} else {
		bodyReader = strings.NewReader("")
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	res, err := s.c.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 && out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return res.StatusCode, data, fmt.Errorf("decode: %w", err)
		}
	}

	return res.StatusCode, data, nil
}

func authMiddleware(token string) fiber.Handler {
	if token == "" {
		// No auth enforced
		return func(c *fiber.Ctx) error { return c.Next() }
	}
	return func(c *fiber.Ctx) error {
		h := c.Get("Authorization")
		const pref = "Bearer "
		if !strings.HasPrefix(h, pref) || strings.TrimPrefix(h, pref) != token {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or missing bearer token")
		}
		return c.Next()
	}
}

func (s *Server) StatusFromErrors(errs map[string]any) int {
	if len(errs) == 0 {
		return fiber.StatusOK
	}
	// If some failed and some succeeded, return 207 Multi-Status-ish (use 207 if behind a proxy that supports it; Fiber doesn’t have const)
	return 207
}
