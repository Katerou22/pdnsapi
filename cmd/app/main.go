package main

import (
	"log"

	"github.com/Katerou22/pdnsapi/internal/record"
	"github.com/Katerou22/pdnsapi/internal/server"
	"github.com/Katerou22/pdnsapi/internal/zone"
	"github.com/Katerou22/pdnsapi/pkg/config"
	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	srvr := server.NewServer(cfg)

	app := srvr.App

	app.Get("/health", func(c *fiber.Ctx) error { return c.SendString("ok") })

	zoneHandler := zone.NewZoneHandler(srvr, cfg)

	zoneHandler.Routes()

	recordHandler := record.NewRecordHandler(srvr, cfg)
	recordHandler.Routes()

	srvr.Run()
}
