package record

import (
	"net/http"
	"strings"

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
	Records []PDNSRrsetChange `json:"rrsets,omitempty"` // when fetching a single zone
}

type PDNSZonePatch struct {
	Kind    string            `json:"kind,omitempty"`
	Account string            `json:"account,omitempty"`
	RRSets  []PDNSRrsetChange `json:"rrsets,omitempty"`
}

type CreateOrUpdateRRSetReq struct {
	Name     string   `json:"name"`               // FQDN with trailing dot
	Type     string   `json:"type"`               // e.g., A, AAAA, CNAME, TXT, MX
	TTL      int      `json:"ttl,omitempty"`      // seconds
	Contents []string `json:"contents"`           // PDNS "content" per record
	Disable  bool     `json:"disabled,omitempty"` // apply same disabled to all records
	// If you pass empty Contents with PATCH, we'll REPLACE to empty (which effectively deletes the rrset).
}

type SimpleRecordReq struct {
	Name  string   `json:"name"`
	Type  []string `json:"type"`
	Value string   `json:"value"`
	TTL   int      `json:"ttl,omitempty"`
}

func NewRecordHandler(srvr *server.Server, c *config.Config) *Handler {

	return &Handler{
		srvr: srvr,
		cfg:  c,
	}
}

func (h *Handler) Routes() {

	app := h.srvr.App

	// Get records (rrsets) for a zone
	app.Get("/zones/:zone/records/", h.records)
	// Create/Replace a record set
	app.Post("/zones/:zone/records/", h.create)

	// Update a record set by recordID = "name:type"
	app.Patch("/zones/:zone/records/:recordID", h.update)

	// Delete a record set by recordID
	app.Delete("/zones/:zone/records/:recordID", h.delete)

	// Simplified record management endpoints
	app.Post("/:zone/create", h.simpleCreate)
	app.Post("/:zone/update", h.simpleUpdate)
	app.Post("/:zone/delete", h.simpleDelete)

}

func (h *Handler) records(ctx *fiber.Ctx) error {

	zone := util.EnsureDot(ctx.Params("zone"))

	cfg := h.cfg
	url := cfg.PDNSURL(cfg.Server, "/zones/"+zone)
	var out PDNSZone
	code, _, err := h.srvr.DoJSON(http.MethodGet, url, cfg.APIKey, nil, &out)
	if err != nil || code >= 300 {
		return ctx.JSON(err)

	}
	return ctx.JSON(out)

}

func (h *Handler) create(ctx *fiber.Ctx) error {

	cfg := h.cfg
	zone := util.EnsureDot(ctx.Params("zone"))
	var req CreateOrUpdateRRSetReq
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.Name == "" || req.Type == "" || len(req.Contents) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "name, type and contents are required")
	}
	rr := PDNSRrsetChange{
		Name:       util.EnsureDot(req.Name),
		Type:       strings.ToUpper(req.Type),
		TTL:        req.TTL,
		ChangeType: "REPLACE",
		Records:    contentsToRecords(req.Contents, req.Disable),
	}
	payload := PDNSZonePatch{RRSets: []PDNSRrsetChange{rr}}

	url := cfg.PDNSURL(cfg.Server, "/zones/"+zone)
	var out any
	code, _, err := h.srvr.DoJSON(http.MethodPatch, url, cfg.APIKey, payload, &out)
	if err != nil || code >= 300 {

		return ctx.JSON(err)

	}
	return ctx.JSON(out)

}
func (h *Handler) update(ctx *fiber.Ctx) error {

	zone := util.EnsureDot(ctx.Params("zone"))
	name, typ, err := util.ParseRecordID(ctx.Params("recordID"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	var req CreateOrUpdateRRSetReq
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	rr := PDNSRrsetChange{
		Name:       name,
		Type:       typ,
		TTL:        req.TTL,
		ChangeType: "REPLACE",
		Records:    contentsToRecords(req.Contents, req.Disable),
	}
	payload := PDNSZonePatch{RRSets: []PDNSRrsetChange{rr}}

	cfg := h.cfg
	url := cfg.PDNSURL(cfg.Server, "/zones/"+zone)
	var out any
	code, _, err := h.srvr.DoJSON(http.MethodPatch, url, cfg.APIKey, payload, &out)
	if err != nil || code >= 300 {
		return ctx.JSON(err)

	}
	return ctx.JSON(out)

}

func (h *Handler) delete(ctx *fiber.Ctx) error {
	zone := util.EnsureDot(ctx.Params("zone"))
	name, typ, err := util.ParseRecordID(ctx.Params("recordID"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	rr := PDNSRrsetChange{
		Name:       name,
		Type:       typ,
		ChangeType: "DELETE",
	}
	payload := PDNSZonePatch{RRSets: []PDNSRrsetChange{rr}}
	cfg := h.cfg

	url := cfg.PDNSURL(cfg.Server, "/zones/"+zone)
	var out any
	code, _, err := h.srvr.DoJSON(http.MethodPatch, url, cfg.APIKey, payload, &out)
	if err != nil || code >= 300 {
		return ctx.JSON(err)

	}
	return ctx.JSON(out)

}

func contentsToRecords(contents []string, disabled bool) []PDNSRecord {
	records := make([]PDNSRecord, 0, len(contents))
	for _, c := range contents {
		records = append(records, PDNSRecord{Content: c, Disabled: disabled})
	}
	return records
}

func (h *Handler) simpleCreate(ctx *fiber.Ctx) error {
	zone := util.EnsureDot(ctx.Params("zone"))
	var req SimpleRecordReq
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.Name == "" || len(req.Type) == 0 || req.Value == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name, type and value are required")
	}

	rrsets := make([]PDNSRrsetChange, 0, len(req.Type))
	for _, t := range req.Type {
		rr := PDNSRrsetChange{
			Name:       util.EnsureDot(req.Name),
			Type:       strings.ToUpper(t),
			TTL:        req.TTL,
			ChangeType: "REPLACE",
			Records:    []PDNSRecord{{Content: req.Value, Disabled: false}},
		}
		rrsets = append(rrsets, rr)
	}

	payload := PDNSZonePatch{RRSets: rrsets}
	url := h.cfg.PDNSURL(h.cfg.Server, "/zones/"+zone)
	var out any
	code, data, err := h.srvr.DoJSON(http.MethodPatch, url, h.cfg.APIKey, payload, &out)
	if err != nil || code >= 300 {
		return ctx.Status(code).SendString(string(data))
	}
	return ctx.JSON(fiber.Map{"status": "created", "zone": zone, "name": req.Name, "types": req.Type})
}

func (h *Handler) simpleUpdate(ctx *fiber.Ctx) error {
	zone := util.EnsureDot(ctx.Params("zone"))
	var req SimpleRecordReq
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.Name == "" || len(req.Type) == 0 || req.Value == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name, type and value are required")
	}

	rrsets := make([]PDNSRrsetChange, 0, len(req.Type))
	for _, t := range req.Type {
		rr := PDNSRrsetChange{
			Name:       util.EnsureDot(req.Name),
			Type:       strings.ToUpper(t),
			TTL:        req.TTL,
			ChangeType: "REPLACE",
			Records:    []PDNSRecord{{Content: req.Value, Disabled: false}},
		}
		rrsets = append(rrsets, rr)
	}

	payload := PDNSZonePatch{RRSets: rrsets}
	url := h.cfg.PDNSURL(h.cfg.Server, "/zones/"+zone)
	var out any
	code, data, err := h.srvr.DoJSON(http.MethodPatch, url, h.cfg.APIKey, payload, &out)
	if err != nil || code >= 300 {
		return ctx.Status(code).SendString(string(data))
	}
	return ctx.JSON(fiber.Map{"status": "updated", "zone": zone, "name": req.Name, "types": req.Type})
}

func (h *Handler) simpleDelete(ctx *fiber.Ctx) error {
	zone := util.EnsureDot(ctx.Params("zone"))
	var req SimpleRecordReq
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.Name == "" || len(req.Type) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "name and type are required")
	}

	rrsets := make([]PDNSRrsetChange, 0, len(req.Type))
	for _, t := range req.Type {
		rr := PDNSRrsetChange{
			Name:       util.EnsureDot(req.Name),
			Type:       strings.ToUpper(t),
			ChangeType: "DELETE",
		}
		rrsets = append(rrsets, rr)
	}

	payload := PDNSZonePatch{RRSets: rrsets}
	url := h.cfg.PDNSURL(h.cfg.Server, "/zones/"+zone)
	var out any
	code, data, err := h.srvr.DoJSON(http.MethodPatch, url, h.cfg.APIKey, payload, &out)
	if err != nil || code >= 300 {
		return ctx.Status(code).SendString(string(data))
	}
	return ctx.JSON(fiber.Map{"status": "deleted", "zone": zone, "name": req.Name, "types": req.Type})
}
