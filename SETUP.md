# Setup and usage

## Prerequisites

Docker Desktop with the Docker service running and Compose v2; or Go 1.25+ and Node.js 22.12+ / npm for native use. Python 3 is optional for the demo. Ports 8080 and 5173 must be free.

## Docker

From this directory:

```sh
cp .env.example .env
docker compose up --build
```

Open http://localhost:5173. The backend health endpoint is http://localhost:8080/health. SQLite persists in the `geofence-data` volume across container restarts. `docker compose down` stops the services and preserves the data. Do not remove the volume unless you intend to erase the database.

## Native development

Terminal 1:

```sh
cd backend
go mod download
go run .
```

Terminal 2:

```sh
cd frontend
npm ci
npm run dev
```

Backend configuration: `DATABASE_URL` (optional PostgreSQL URL, overrides SQLite), `PORT` (8080), `DATABASE_PATH` (`./data/geofence.db`), `CORS_ORIGIN` (`http://localhost:5173`, exact origin with no trailing slash). Set these in the process environment. The native backend does not automatically load `.env`.

Frontend configuration: create `frontend/.env.local` containing `VITE_API_URL=http://localhost:8080`. This URL must be accessible from the user's browser, not a Docker internal hostname. Restart Vite after changing it. Production values are embedded at build time.

## Dashboard workflow

1. Geofences: enter name, description and category. Draw at least three distinct vertices on the map, then choose **Close polygon**. Alternatively paste a closed array of `[latitude, longitude]` pairs. Save.
2. Vehicles: register vehicle number, driver, type and phone. Duplicate numbers return 409.
3. Alert rules: select a fence, one or all vehicles, and entry, exit or both. Activate the rule before moving the vehicle.
4. Vehicles: select the vehicle and click **Choose on map**, or enter coordinates manually. Set a new timestamp and update location. A first report inside is an entry; subsequent reports trigger events only when membership changes.
5. Open a second browser tab. Move the vehicle outside, inside, and outside again with increasing timestamps. Both tabs receive the same matching alerts.
6. Event history: filter by vehicle, fence and date range; page through records. History includes all transitions, even those with no matching alert rule. The live feed displays matching alerts received during the current browser session.

Fleet positions refresh every 10 seconds and immediately following local actions or matching alerts. The map defaults to San Francisco; use **Fit boundaries** to see fences elsewhere. Manual map inputs do not use the browser's location permission.

## Repeatable demonstration

With the API and dashboard running:

```sh
python3 scripts/demo.py
```

This adds a demo fence, a uniquely numbered vehicle and a rule, then emits entry and exit events. It adds data on every run.

## API testing examples

```sh
curl -s http://localhost:8080/health
curl -s -X POST http://localhost:8080/geofences \
  -H 'Content-Type: application/json' \
  -d '{"name":"Depot","description":"Restricted loading area","category":"restricted_zone","coordinates":[[37.77,-122.43],[37.785,-122.43],[37.785,-122.41],[37.77,-122.41],[37.77,-122.43]]}'
curl -s -X POST http://localhost:8080/vehicles \
  -H 'Content-Type: application/json' \
  -d '{"vehicle_number":"KA-01-AB-1234","driver_name":"Demo Driver","vehicle_type":"truck","phone":"+10000000000"}'
```

Copy the returned IDs into `GEOFENCE_ID` and `VEHICLE_ID` below.

```sh
GEOFENCE_ID='paste-returned-geofence-id'
VEHICLE_ID='paste-returned-vehicle-id'
curl -s -X POST http://localhost:8080/alerts/configure \
  -H 'Content-Type: application/json' \
  -d "{\"geofence_id\":\"$GEOFENCE_ID\",\"vehicle_id\":\"$VEHICLE_ID\",\"event_type\":\"both\"}"
curl -s -X POST http://localhost:8080/vehicles/location \
  -H 'Content-Type: application/json' \
  -d "{\"vehicle_id\":\"$VEHICLE_ID\",\"latitude\":37.777,\"longitude\":-122.42,\"timestamp\":\"2026-11-13T10:00:00Z\"}"
curl -s "http://localhost:8080/vehicles/location/$VEHICLE_ID"
curl -s 'http://localhost:8080/geofences?category=restricted_zone'
curl -s http://localhost:8080/vehicles
curl -s "http://localhost:8080/alerts?geofence_id=$GEOFENCE_ID"
curl -s "http://localhost:8080/violations/history?vehicle_id=$VEHICLE_ID&limit=10&offset=0"
```

Use newer timestamps for subsequent updates. Date filters accept URL-encoded RFC3339 `start_date` and `end_date`, inclusive. History defaults to 50 records; limit is 1–500, offset defaults to 0. `total_count` is the matching count before pagination. Invalid requests return an `error` string and `time_ns`.

## Checks

```sh
cd backend
go test -race ./...
go vet ./...
cd ../frontend
npm ci
npm run build
cd ..
docker compose config --quiet
```

Tests exercise validation, geometry and boundary behavior, all required endpoints, duplicate registration, stale timestamps, persistence, history pagination and identical entry/exit delivery to two WebSocket clients.

## Backend deployment and Docker Hub

Choose your own Docker Hub namespace and a Docker-capable hosting service with a persistent disk. From the project root:

```sh
docker buildx build --platform linux/amd64,linux/arm64 \
  -t YOUR_DOCKERHUB_USERNAME/geofence-api:1.0.0 --push backend
```

Sign in to Docker Hub first using your own account. Deploy that image with port 8080 exposed, `PORT=8080`, `DATABASE_PATH=/data/geofence.db`, a persistent disk mounted at `/data`, and `CORS_ORIGIN=https://YOUR-FRONTEND-DOMAIN`. Use one backend instance with this SQLite design. Your hosting proxy must support WebSocket upgrades and HTTPS. Verify `/health` and save the real backend URL.

## Frontend deployment

For Vercel, Netlify or Cloudflare Pages, choose `frontend` as the root directory, `npm ci && npm run build` as the build command and `dist` as the output directory. Set `VITE_API_URL=https://YOUR-BACKEND-DOMAIN` before building. The app derives `wss://` automatically from HTTPS. Update backend `CORS_ORIGIN` to the final frontend origin. Test creation, updates and live alerts against the deployed API in two browser tabs.

For Docker-hosted frontend deployments, pass `--build-arg VITE_API_URL=https://YOUR-BACKEND-DOMAIN` when building `frontend`. Setting this variable only at container runtime will not change the bundled frontend.

## Storage and delivery architecture

SQLite tables are created on boot: fences, vehicles (unique number), locations (append-only with per-vehicle index), rules and events (history JSON and optional matched alert JSON). A single mutex orders REST requests; a location and its events are committed together. Alerts enter per-client buffered channels only after commit. Network writes run in each WebSocket handler, outside the REST handler; full queues disconnect slow clients. The live delivery mechanism is best-effort: a crash after commit can leave an event stored without live delivery. Add an outbox dispatcher for guaranteed retries.

## Troubleshooting

- Docker connection error: start Docker Desktop and wait for its service to be ready.
- Fetch errors: check the API process, `VITE_API_URL`, HTTPS and exact CORS origin.
- No alerts: verify an active matching rule exists, coordinates actually cross the fence, and timestamp increases. Merely remaining inside is not another entry.
- Missing map tiles/fonts: check internet access. API and coordinate forms do not depend on map tile availability.
- Deployment restart loses data: mount a persistent disk at the configured database path.

## PostgreSQL and free Render hosting

Set `DATABASE_URL` to a PostgreSQL connection URL to select the remote database adapter. When omitted, SQLite remains the local default. The included `render.yaml` provisions free-only API, PostgreSQL and static frontend resources. Render free PostgreSQL expires after 30 days; free web services may sleep. Check the limitations in DEPLOYMENT.md before using it for longer-term hosting.

To run the same integration suite against your own disposable PostgreSQL test database, set `TEST_DATABASE_URL` before `go test -race ./...`. Each test run creates and removes an isolated generated schema. Do not grant the test process access to production data.
