# PDNS API



## Environment Variables

| Variable         | Required | Example                                                 | Notes                                                           |
|------------------| -------- |---------------------------------------------------------|-----------------------------------------------------------------|
| `PDNS_SERVER`    | ✅        | `http://pdns1:8081/api/v1`  **Must include** `/api/v1`. |
| `PDNS_APIKEY`    | ❌        | `secret`                                                | .                                                               |
| `PDNS_SERVER_ID` | ❌        | `localhost`                                             | PDNS server-id path segment. Defaults to `localhost`.           |
| `AUTH_TOKEN`     | ❌        | `supersecret`                                           | If set, clients must send `Authorization: Bearer <AUTH_TOKEN>`. |
| `ADDR`           | ❌        | `:8080`                                                 | Listen address for the app.                                     |

---

## Authentication

If `AUTH_TOKEN` is set, **all requests** must include:

```
Authorization: Bearer <AUTH_TOKEN>
```

---

## Conventions & Notes

* **FQDNs with trailing dot**: Use trailing dots for zones and record names (e.g., `example.com.`).
  The app auto-appends a dot if missing.
* **recordID format**: `name:type` (e.g., `www.example.com.:A`).


## Endpoints

### 1) Create Zone

**POST** `/zones`

Create a zone.

**Request body**

```json
{
  "name": "example.com.",
  "kind": "Native",
  "masters": ["203.0.113.10"],
  "dnssec": false,
  "account": "owner-id",
  "nameservers": ["ns1.example.com.", "ns2.example.com."],
  "rrsets": [
    {
      "name": "example.com.",
      "type": "SOA",
      "ttl": 3600,
      "changetype": "REPLACE",
      "records": [
        { "content": "ns1.example.com. hostmaster.example.com. 1 10800 3600 604800 3600", "disabled": false }
      ]
    }
  ]
}
```

**Response (200 or 207)**

```json
{
  "id": "…",
  "name": "example.com.",
  "kind": "Native"
}
```

---

### 2) Update Zone

**PATCH** `/zones/:zoneName`

Update zone properties and/or apply RRset changes.

> `:zoneName` should be an FQDN (trailing dot optional; app will normalize).

**Request body**

```json
{
  "kind": "Native",
  "account": "owner-id",
  "rrsets": [
    {
      "name": "www.example.com.",
      "type": "A",
      "ttl": 120,
      "changetype": "REPLACE",
      "records": [
        { "content": "203.0.113.42", "disabled": false }
      ]
    }
  ]
}
```



### 3) List Zones

**GET** `/zones`

Aggregates zones from each PDNS server.

**Response**

```json
[
  {
    "id": "…",
    "name": "example.org.",
    "kind": "Native"
  },
  ...
]
```

---

### 4) List Zone Records (RRsets)

**GET** `/zone/:zoneName/records`

Returns RRsets for the zone (one list per server).

**Response**

```json
[
  {
    "name": "example.com.",
    "type": "SOA",
    "ttl": 3600,
    "records": [
      {
        "content": "…",
        "disabled": false
      }
    ]
  },
  {
    "name": "www.example.com.",
    "type": "A",
    "ttl": 120,
    "records": [
      {
        "content": "203.0.113.42",
        "disabled": false
      }
    ],
    ...
  ]
```

---

### 5) Create / Replace a Record Set

**POST** `/zone/:zoneName/records`

Create or replace a single RRset in the zone.

**Request body**

```json
{
  "name": "www.example.com.",
  "type": "A",
  "ttl": 300,
  "contents": ["203.0.113.10", "203.0.113.11"],
  "disabled": false
}
```



---

### 6) Update a Record Set (by recordID)

**PATCH** `/zone/:zoneName/records/:recordID`

* `:recordID` format: `name:type` (e.g., `www.example.com.:A`)

**Request body**

```json
{
  "ttl": 120,
  "contents": ["203.0.113.12"],
  "disabled": false
}
```



---

### 7) Delete a Record Set (by recordID)

**DELETE** `/zone/:zoneName/records/:recordID`

Deletes the entire RRset (`name:type`).



---

## Record Content Reference

Common RRtypes and expected `contents` strings:

* **A**: IPv4, e.g. `"203.0.113.10"`
* **AAAA**: IPv6, e.g. `"2001:db8::1"`
* **CNAME**: Canonical host **with trailing dot**, e.g. `"target.example.net."`
* **MX**: `"10 mail.example.com."` (priority + host with trailing dot)
* **TXT**: Raw string; PDNS handles quoting, e.g. `"v=spf1 include:_spf.example.com ~all"`
* **NS**: Nameserver host **with trailing dot**, e.g. `"ns1.example.com."`
* **SRV**: `"10 5 443 service.example.com."` (priority weight port target.)
* **CAA**: `"0 issue \"letsencrypt.org\""` (full RFC content as a single string)

> The app converts `contents[]` into PDNS `records[]` with the same `disabled` flag applied to each item.

---

## Errors & Partial Success

### Validation Errors

* `400 Bad Request` with a message string.

### Authorization Missing/Invalid

* `401 Unauthorized` if `AUTH_TOKEN` is set and missing/invalid.


## cURL Examples

Create zone:

```bash
curl -X POST "$BASE/zones" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"example.com.","kind":"Native"}'
```

List zones:

```bash
curl -H "Authorization: Bearer $TOKEN" "$BASE/zones"
```

Create A record:

```bash
curl -X POST "$BASE/zone/example.com./records" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"www.example.com.","type":"A","ttl":300,"contents":["203.0.113.10"]}'
```

Update A record via recordID:

```bash
curl -X PATCH "$BASE/zone/example.com./records/www.example.com.:A" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ttl":120,"contents":["203.0.113.11"]}'
```

Delete A record:

```bash
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  "$BASE/zone/example.com./records/www.example.com.:A"
```

