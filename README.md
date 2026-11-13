# Boundary — Geofencing & Real-time Alert System

A Go API and React fleet dashboard with interactive Leaflet maps, persistent SQLite storage, and WebSocket entry/exit alerts. This is an original implementation created with AI assistance; review and understand the code before presenting it.

## Run

```sh
docker compose up --build
```

Open http://localhost:5173. API: http://localhost:8080. See [SETUP.md](SETUP.md) for native setup, deployment, API examples and verification.

## Included

- All nine required REST endpoints and `/ws/alerts`.
- Closed polygon validation, category/range checks, self-intersection rejection, boundary-inclusive point containment.
- Vehicle registration, map and manual location inputs, rules for one or all vehicles.
- First observation inside counts as entry; repeated inside reports do not repeat alerts. Older or equal timestamps return 409.
- Every JSON API response includes a string `time_ns`, including errors.
- Live alerts for multiple clients, automatic browser reconnect and periodic fleet refresh.
- All transitions and matching alert payloads saved transactionally; history has vehicle, geofence, date filters and offset pagination.
- Responsive dashboard, category filters, loading/error/success feedback, Dockerfiles, Compose, integration tests and demo script.

## Project structure

```text
backend/       Go REST API, geometry, SQLite schema, WebSocket delivery, tests
frontend/      React dashboard, Leaflet map, Vite configuration
scripts/       Repeatable demo data and vehicle movement
SETUP.md       Local use, API examples and deployment instructions
SUBMISSION.md  Form fields and publication checklist
VIDEO_GUIDE.md 35–40 minute recording outline
```

## Submission links — replace after publishing

| Required item | Status |
| --- | --- |
| Private solution GitHub repository | Not published |
| Docker Hub backend image | Not published |
| Deployed frontend | Not deployed |
| Deployed backend | Not deployed |
| 30–45 minute solution video | Not recorded |

No published links or completed submission are claimed. The ZIP contains source code and instructions; external publication and the personal walkthrough remain separate steps.

## Design choices and limits

SQLite provides a real embedded database without a separate service; Docker uses a named persistent volume. One process serializes requests so location comparisons and transactions cannot race. WebSocket clients have bounded queues, write deadlines and heartbeat checks; slow clients disconnect and can reconnect. History remains available after reconnect, but the transient live feed is not replayed automatically.

This is a small-scale assessment implementation, not a production fleet platform. Polygon containment scans all fences; history filtering scans stored events; there is no spatial index. Polygons are planar and intended for local areas, not antimeridian/polar regions. There is no authentication, tenant isolation, rate limiting, rule editing/deletion, or horizontal WebSocket distribution. CORS restricts browser origins, not API access. Add authentication before storing real fleet data or exposing it to untrusted users.

For scale: use PostGIS with a GiST spatial index, per-vehicle transaction locks, SQL-side indexed history filters and keyset pagination, an outbox and broker for reliable cross-process alert delivery, authentication, observability, backups and a tile provider appropriate for your traffic.

Dependencies are pinned in Go and npm lockfiles. Map tiles and fonts require internet access. OSM attribution remains visible.
