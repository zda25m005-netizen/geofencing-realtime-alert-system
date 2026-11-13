# Boundary — Geofencing & Real-time Alert System

A Go API and React fleet dashboard with interactive Leaflet maps, SQLite/PostgreSQL storage, and WebSocket entry/exit alerts. This is an original implementation created with AI assistance; review and understand the code before presenting it.

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
| Private solution GitHub repository | [zda25m005-netizen/geofencing-realtime-alert-system](https://github.com/zda25m005-netizen/geofencing-realtime-alert-system) |
| Docker Hub backend image | [jhag8094/geofence-api:1.0.0](https://hub.docker.com/r/jhag8094/geofence-api/tags?name=1.0.0) |
| Deployed frontend | Not deployed |
| Deployed backend | Not deployed |
| 30–45 minute solution video | 36:30 AI-narrated walkthrough generated; unlisted hosting pending |

The private source repository is published. The backend image is published on Docker Hub for Linux AMD64 and ARM64. Hosted services and an unlisted hosted video remain pending. The 36:30 AI-narrated video is supplied separately from the source ZIP and as a private release asset. Its unlisted YouTube/Loom link must be added after upload. [Publication status](PUBLICATION_STATUS.md) lists each remaining step.

## Design choices and limits

SQLite provides an embedded database for local use; Docker uses a named persistent volume. Set `DATABASE_URL` to a PostgreSQL URL for remote persistence on free hosting. `backend/database.go` adapts the repository-owned parameterized SQL for both stores. PostgreSQL does not yet use PostGIS. One process serializes requests so location comparisons and transactions cannot race. WebSocket clients have bounded queues, write deadlines and heartbeat checks; slow clients disconnect and can reconnect. History remains available after reconnect, but the transient live feed is not replayed automatically.

This is a small-scale assessment implementation, not a production fleet platform. Polygon containment scans all fences; history filtering scans stored events; there is no spatial index. Polygons are planar and intended for local areas, not antimeridian/polar regions. There is no authentication, tenant isolation, rate limiting, rule editing/deletion, or horizontal WebSocket distribution. CORS restricts browser origins, not API access. Add authentication before storing real fleet data or exposing it to untrusted users.

For scale: use PostGIS with a GiST spatial index, per-vehicle transaction locks, SQL-side indexed history filters and keyset pagination, an outbox and broker for reliable cross-process alert delivery, authentication, observability, backups and a tile provider appropriate for your traffic.

Dependencies are pinned in Go and npm lockfiles. Map tiles and fonts require internet access. OSM attribution remains visible.

## Free hosting

`render.yaml` uses only free plans: a Docker API, remote PostgreSQL, and a static dashboard. Free Render PostgreSQL expires after 30 days; web services can sleep on inactivity. No paid upgrades are authorized. See [DEPLOYMENT.md](DEPLOYMENT.md).

## Prepared walkthrough

[Download the AI-narrated video from the private release](https://github.com/zda25m005-netizen/geofencing-realtime-alert-system/releases/download/v1.0.0/boundary-ai-walkthrough.mp4). This download is not an unlisted YouTube or Loom link. [Transcript](docs/AI_WALKTHROUGH_TRANSCRIPT.md). Narration is synthetic and must not be represented as the candidate speaking personally.

## Published backend image

`docker pull jhag8094/geofence-api:1.0.0`

[Docker Hub image](https://hub.docker.com/r/jhag8094/geofence-api). Verified multi-platform digest: `sha256:cf073f133b45e3216478db933dce77882ec30c0ffa299e207a0af04c82139c6f`.
