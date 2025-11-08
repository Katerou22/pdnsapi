package zone

import (
	"net/http"

	"github.com/Katerou22/pdnsapi/internal/server"
	"github.com/Katerou22/pdnsapi/pkg/config"
	"github.com/Katerou22/pdnsapi/pkg/util"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	srvr *server.Server
	cfg  *config.Config
}

type CreateZoneReq struct {
	Name        string            `json:"name"`                  // example.com. (FQDN, trailing dot)
	Kind        string            `json:"kind,omitempty"`        // Native, Master, Slave (default: Native)
	Masters     []string          `json:"masters,omitempty"`     // For Slave zones
	DNSSec      bool              `json:"dnssec,omitempty"`      // Optional
	Account     string            `json:"account,omitempty"`     // Optional
	Nameservers []string          `json:"nameservers,omitempty"` // Optional
	RRSets      []PDNSRrsetChange `json:"rrsets,omitempty"`      // Optional initial rrsets
}

type PDNSZoneCreate struct {
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	Masters     []string          `json:"masters,omitempty"`
	DNSSec      bool              `json:"dnssec,omitempty"`
	Account     string            `json:"account,omitempty"`
	Nameservers []string          `json:"nameservers,omitempty"`
	RRSets      []PDNSRrsetChange `json:"rrsets,omitempty"`
}
type UpdateZoneReq struct {
	Kind    string            `json:"kind,omitempty"`
	Account string            `json:"account,omitempty"`
	RRSets  []PDNSRrsetChange `json:"rrsets,omitempty"`
}

type PDNSRecord struct {
	Content  string `json:"content"`
	Disabled bool   `json:"disabled"`
}

type PDNSRrsetChange struct {
	Name       string       `json:"name"` // FQDN with trailing dot
	Type       string       `json:"type"`
	TTL        int          `json:"ttl,omitempty"`
	ChangeType string       `json:"changetype,omitempty"` // "REPLACE" or "DELETE"
	Records    []PDNSRecord `json:"records,omitempty"`
}

type PDNSZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	// Many more fields exist in PDNS; we only surface a subset to keep it simple.
	//RRSets []PDNSRrsetChange `json:"rrsets,omitempty"` // when fetching a single zone
}

type PDNSZonePatch struct {
	Kind    string            `json:"kind,omitempty"`
	Account string            `json:"account,omitempty"`
	RRSets  []PDNSRrsetChange `json:"rrsets,omitempty"`
}

func NewZoneHandler(srvr *server.Server, c *config.Config) *Handler {

	return &Handler{
		srvr: srvr,
		cfg:  c,
	}
}

func (h *Handler) Routes() {

	app := h.srvr.App
	// Create zone
	app.Post("/zones", h.create)

	// Update zone (PATCH rrsets / kind / account)
	app.Patch("/zones/:zone", h.update)

	// List zones
	app.Get("/zones", h.list)
}

func (h *Handler) create(ctx *fiber.Ctx) error {

	cfg := h.cfg
	var req CreateZoneReq
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required (FQDN, include trailing dot or we'll add it)")
	}
	if req.Kind == "" {
		req.Kind = "Native"
	}

	payload := PDNSZoneCreate{
		Name:        util.EnsureDot(req.Name),
		Kind:        req.Kind,
		Masters:     req.Masters,
		DNSSec:      req.DNSSec,
		Account:     req.Account,
		Nameservers: req.Nameservers,
		RRSets:      req.RRSets,
	}

	url := cfg.PDNSURL(cfg.Server, "/zones")
	var out PDNSZone
	code, _, err := h.srvr.DoJSON(http.MethodPost, url, cfg.APIKey, payload, &out)
	if err != nil || code >= 300 {
		if err != nil {

			return ctx.JSON(err)

		}
	}

	return ctx.JSON(out)

}

func (h *Handler) update(ctx *fiber.Ctx) error {
	cfg := h.cfg

	zone := util.EnsureDot(ctx.Params("zone"))
	var req UpdateZoneReq
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	payload := PDNSZonePatch{
		Kind:    req.Kind,
		Account: req.Account,
		RRSets:  req.RRSets,
	}

	url := cfg.PDNSURL(cfg.Server, "/zones/"+zone)
	var out any
	code, _, err := h.srvr.DoJSON(http.MethodPatch, url, cfg.APIKey, payload, &out)
	if err != nil || code >= 300 {
		if err != nil {
			return ctx.JSON(err)

		}
	}
	return ctx.JSON(out)
}

func (h *Handler) list(ctx *fiber.Ctx) error {
	cfg := h.cfg

	url := cfg.PDNSURL(cfg.Server, "/zones")
	var out []PDNSZone
	code, _, err := h.srvr.DoJSON(http.MethodGet, url, cfg.APIKey, nil, &out)
	if err != nil || code >= 300 {
		return ctx.JSON(err)

	}
	return ctx.JSON(out)
}
