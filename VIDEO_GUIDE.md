# Personal walkthrough outline — 35–40 minutes

Use this as an outline while explaining the actual code in your own words. Record your own screen and narration; the package does not include a recorded video.

## 0:00–11:00 — Working demonstration

- Show the dashboard and explain geofences, vehicles, rules and event history.
- Draw and save a closed restricted-zone polygon. Show an invalid polygon being rejected.
- Register a vehicle and configure entry/exit for it, or all vehicles.
- Open a second browser tab. Send outside, inside, inside again and outside updates with newer timestamps.
- Show live alerts in both tabs, with no duplicate alert while remaining inside.
- Show location and geofence status, category filtering and history filters/pagination.
- Show Docker Compose starting locally and data surviving a restart. Only claim this when you have actually run it.

## 11:00–26:00 — Implementation walkthrough

- Start at backend/main.go: App state, schema initialization and routes.
- Trace a vehicle location request: input validation, latest location, polygon checks, transition detection, matching rules, transaction commit and buffered delivery.
- Explain [latitude, longitude], ray casting, boundary inclusion, closed polygons and validation tests.
- Show the SQLite tables and indexes. Explain why it was chosen and where scans limit scale.
- Explain serialization, WebSocket client locking, bounded queues, ping/pong and slow-client disconnection.
- Show main_test.go and run `go test -race ./...`.
- Show frontend/src/main.jsx: API calls, forms, map layers, client reconnect, state and history pagination.
- Explain the Docker stages, persistent volume, CORS and build-time API URL.

## 26:00–38:00 — Tradeoffs and roadmap

- State AI assistance honestly and explain what you verified yourself.
- Explain first-location entry behavior, boundary points, stale timestamps and duplicate rule deduplication.
- Explain current limits: planar polygons, all-fence scans, history scans, serialized requests, no authentication, no rule editing and no automatic alert replay.
- Describe an approximate plan: authentication/tenant isolation (1–2 days), PostGIS and indexed queries (2–3 days), outbox/broker and multi-instance delivery (2–3 days), load/operational testing (1–2 days). These are planning estimates, not guarantees.
- Describe frontend static asset hashing and why rebuilding is needed for API URL changes.
- Finish by showing your actual deployment and repository links.
