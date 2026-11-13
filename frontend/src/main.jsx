import React, { useEffect, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import "./style.css";
const API = (import.meta.env.VITE_API_URL || "http://localhost:8080").replace(
  /\/$/,
  "",
);
const categories = [
  "delivery_zone",
  "restricted_zone",
  "toll_zone",
  "customer_area",
];
const label = (s) => s?.replaceAll("_", " ") || "—";
async function request(path, data) {
  const r = await fetch(
    API + path,
    data
      ? {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(data),
        }
      : {},
  );
  const body = await r.json();
  if (!r.ok) throw Error(body.error || "Request failed");
  return body;
}
function App() {
  const [tab, setTab] = useState("Overview"),
    [fences, setFences] = useState([]),
    [vehicles, setVehicles] = useState([]),
    [rules, setRules] = useState([]),
    [positions, setPositions] = useState([]),
    [feed, setFeed] = useState([]),
    [connected, setConnected] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [mode, setMode] = useState("inspect"),
    [draft, setDraft] = useState([]),
    [selected, setSelected] = useState(""),
    [point, setPoint] = useState([37.7749, -122.4194]),
    [category, setCategory] = useState(""),
    [history, setHistory] = useState([]),
    [total, setTotal] = useState(0),
    [offset, setOffset] = useState(0),
    [filters, setFilters] = useState({
      vehicle_id: "",
      geofence_id: "",
      start_date: "",
      end_date: "",
    });
  const mapEl = useRef(),
    map = useRef(),
    layers = useRef(),
    clickState = useRef();
  clickState.current = { mode, setDraft, setPoint };
  async function refresh() {
    const [f, v, r] = await Promise.all([
      request("/geofences"),
      request("/vehicles"),
      request("/alerts"),
    ]);
    setFences(f.geofences);
    setVehicles(v.vehicles);
    setRules(r.alerts);
    setPositions(
      await Promise.all(
        v.vehicles.map((x) => request("/vehicles/location/" + x.id)),
      ),
    );
  }
  async function loadHistory() {
    const q = new URLSearchParams({ limit: 10, offset });
    for (const [k, v] of Object.entries(filters))
      if (v) q.set(k, k.endsWith("_date") ? new Date(v).toISOString() : v);
    const h = await request("/violations/history?" + q);
    setHistory(h.violations);
    setTotal(h.total_count);
  }
  useEffect(() => {
    refresh().catch((e) => setError(e.message));
    const timer = setInterval(
      () => refresh().catch((e) => setError(e.message)),
      10000,
    );
    return () => clearInterval(timer);
  }, []);
  useEffect(() => {
    loadHistory().catch((e) => setError(e.message));
  }, [filters, offset, tab]);
  useEffect(() => {
    let closed = false,
      timer,
      ws;
    const connect = () => {
      ws = new WebSocket(API.replace(/^http/, "ws") + "/ws/alerts");
      ws.onopen = () => setConnected(true);
      ws.onmessage = (e) => {
        try {
          const item = JSON.parse(e.data);
          setFeed((old) => [item, ...old].slice(0, 100));
          setNotice(
            `${item.vehicle.vehicle_number}: ${item.event_type} at ${item.geofence.geofence_name}`,
          );
          refresh().catch((e) => setError(e.message));
        } catch {
          setError("Could not read a live alert");
        }
      };
      ws.onerror = () => ws.close();
      ws.onclose = () => {
        setConnected(false);
        if (!closed) timer = setTimeout(connect, 2500);
      };
    };
    connect();
    return () => {
      closed = true;
      clearTimeout(timer);
      ws?.close();
    };
  }, []);
  useEffect(() => {
    if (!mapEl.current) return;
    map.current = L.map(mapEl.current).setView([37.7749, -122.4194], 13);
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "© OpenStreetMap contributors",
      maxZoom: 19,
    }).addTo(map.current);
    layers.current = L.layerGroup().addTo(map.current);
    map.current.on("click", (e) => {
      const { mode, setDraft, setPoint } = clickState.current;
      const p = [
        Number(e.latlng.lat.toFixed(6)),
        Number(e.latlng.lng.toFixed(6)),
      ];
      if (mode === "draw") setDraft((old) => [...old, p]);
      if (mode === "move") setPoint(p);
    });
    return () => map.current.remove();
  }, []);
  useEffect(() => {
    if (!layers.current) return;
    layers.current.clearLayers();
    for (const f of fences.filter(
      (f) => !category || f.category === category,
    )) {
      const color = f.category === "restricted_zone" ? "#dc694f" : "#327863";
      const polygon = L.polygon(f.coordinates, {
        color,
        weight: 2,
        fillOpacity: 0.16,
      }).addTo(layers.current);
      polygon.bindTooltip(document.createTextNode(f.name));
    }
    if (draft.length)
      L.polyline(draft, { color: "#d39632", dashArray: "6 6" }).addTo(
        layers.current,
      );
    for (const p of positions) {
      if (!p.current_location) continue;
      const loc = p.current_location;
      const marker = L.circleMarker([loc.latitude, loc.longitude], {
        radius: 9,
        color: "#fff",
        weight: 3,
        fillColor: "#143e31",
        fillOpacity: 1,
      }).addTo(layers.current);
      marker.bindTooltip(document.createTextNode(p.vehicle_number));
    }
    if (mode === "move")
      L.circleMarker(point, { radius: 8, color: "#d39632" }).addTo(
        layers.current,
      );
  }, [fences, positions, draft, mode, point, category]);
  useEffect(() => {
    setTimeout(() => map.current?.invalidateSize(), 50);
  }, [tab]);
  async function action(fn, message) {
    setBusy(true);
    setError("");
    try {
      await fn();
      await refresh();
      await loadHistory();
      setNotice(message);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }
  function submit(e, path, transform = (x) => x, after = () => {}) {
    e.preventDefault();
    const form = e.currentTarget,
      data = Object.fromEntries(new FormData(form));
    action(async () => {
      await request(path, transform(data));
      form.reset();
      after();
    }, "Saved successfully");
  }
  const vehicleOptions = (
    <>
      <option value="">All vehicles</option>
      {vehicles.map((v) => (
        <option key={v.id} value={v.id}>
          {v.vehicle_number}
        </option>
      ))}
    </>
  );
  const fenceOptions = (
    <>
      <option value="">Select geofence</option>
      {fences.map((f) => (
        <option key={f.id} value={f.id}>
          {f.name}
        </option>
      ))}
    </>
  );
  return (
    <div className="shell">
      <aside>
        <a className="brand" href="#">
          ◈{" "}
          <span>
            boundary<span className="brand-dot">.</span>
          </span>
        </a>
        <div className="workspace">
          WORKSPACE{" "}
          <b>
            Fleet operations <span>⌄</span>
          </b>
        </div>
        <nav>
          {[
            "Overview",
            "Geofences",
            "Vehicles",
            "Alert rules",
            "Event history",
          ].map((t, i) => (
            <button
              key={t}
              className={tab === t ? "active" : ""}
              onClick={() => setTab(t)}
            >
              <span>{["▦", "⬡", "▰", "♧", "◷"][i]}</span>
              {t}
              {t === "Alert rules" && <small>{rules.length}</small>}
            </button>
          ))}
        </nav>
        <div className="aside-bottom">
          <span className={"dot " + (connected ? "green" : "")}></span>
          {connected ? "Live connection active" : "Reconnecting…"}
          <p>BOUNDARY / OPERATIONS CONSOLE</p>
        </div>
      </aside>
      <main>
        <header>
          <span>
            Workspace <span className="slash">/</span> {tab}
          </span>
          <div>
            <span className={"dot " + (connected ? "green" : "")}></span>{" "}
            {connected ? "Live updates" : "Offline"}{" "}
            <span className="avatar">OP</span>
          </div>
        </header>
        <section className="content">
          <div className="page-title">
            <div>
              <div className="eyebrow">FLEET INTELLIGENCE</div>
              <h1>{tab === "Overview" ? "Every movement. In view." : tab}</h1>
              <p>
                Monitor your fleet, define boundaries, and act on what matters.
              </p>
            </div>
            <button
              className="primary"
              onClick={() => {
                setTab("Geofences");
                setMode("draw");
              }}
            >
              ＋ Create geofence
            </button>
          </div>
          {error && (
            <div className="error" role="alert">
              {error}
              <button onClick={() => setError("")}>×</button>
            </div>
          )}
          {notice && (
            <div className="notice" role="status">
              ✓ {notice}
              <button onClick={() => setNotice("")}>×</button>
            </div>
          )}
          <div className="stats">
            {[
              [
                "Registered vehicles",
                vehicles.length,
                "Fleet ready to track",
                "▰",
              ],
              [
                "Active geofences",
                fences.length,
                "Your operational boundaries",
                "⬡",
              ],
              ["Alert rules", rules.length, "Monitoring entries & exits", "♧"],
              ["Events recorded", total, "Saved movement transitions", "↗"],
            ].map(([t, n, d, icon]) => (
              <article key={t}>
                <span>
                  {t}
                  <i>{icon}</i>
                </span>
                <strong>{n.toString().padStart(2, "0")}</strong>
                <small>{d}</small>
              </article>
            ))}
          </div>
          <div className="columns">
            <div className="left">
              <section className="card map-card">
                <div className="card-head">
                  <div>
                    <h2>Live operations map</h2>
                    <p>
                      {mode === "draw"
                        ? "Click the map to add polygon vertices."
                        : mode === "move"
                          ? "Click the map to choose a vehicle location."
                          : "Your geofences and latest vehicle positions."}
                    </p>
                  </div>
                  <select
                    aria-label="Filter map by category"
                    value={category}
                    onChange={(e) => setCategory(e.target.value)}
                  >
                    <option value="">All categories</option>
                    {categories.map((c) => (
                      <option key={c}>{c}</option>
                    ))}
                  </select>
                </div>
                <div ref={mapEl} className="map" />
                <div className="map-footer">
                  <span>
                    <i className="legend safe" />
                    Operational zone <i className="legend restricted" />
                    Restricted zone <i className="legend vehicle" />
                    Vehicle
                  </span>
                  <button
                    onClick={() => {
                      const points = fences.flatMap((f) => f.coordinates);
                      if (points.length)
                        map.current.fitBounds(points, { padding: [30, 30] });
                    }}
                  >
                    Fit boundaries ↗
                  </button>
                </div>
              </section>
              {tab === "Overview" && (
                <section className="card">
                  <div className="card-head">
                    <div>
                      <h2>Fleet at a glance</h2>
                      <p>Latest positions and boundary status</p>
                    </div>
                    <button onClick={() => setTab("Vehicles")}>
                      View fleet →
                    </button>
                  </div>
                  <Fleet vehicles={vehicles} positions={positions} />
                </section>
              )}
              {tab === "Geofences" && (
                <section className="card panel">
                  <h2>Create a geofence</h2>
                  <p>
                    Draw vertices on the map or enter [latitude, longitude]
                    coordinates. The final point must match the first.
                  </p>
                  <div className="tools">
                    <button
                      onClick={() =>
                        setMode(mode === "draw" ? "inspect" : "draw")
                      }
                    >
                      {mode === "draw" ? "Stop drawing" : "Draw on map"}
                    </button>
                    <button onClick={() => setDraft((d) => d.slice(0, -1))}>
                      Undo vertex
                    </button>
                    <button onClick={() => setDraft([])}>Clear</button>
                    <button
                      onClick={() =>
                        setDraft((d) => (d.length >= 3 ? [...d, d[0]] : d))
                      }
                    >
                      Close polygon
                    </button>
                  </div>
                  <form
                    onSubmit={(e) =>
                      submit(
                        e,
                        "/geofences",
                        (d) => ({
                          ...d,
                          coordinates: JSON.parse(d.coordinates),
                        }),
                        () => {
                          setDraft([]);
                          setMode("inspect");
                        },
                      )
                    }
                  >
                    <div className="form-grid">
                      <Field
                        label="Geofence name"
                        name="name"
                        placeholder="Downtown delivery zone"
                      />
                      <label>
                        Category
                        <select name="category">
                          {categories.map((c) => (
                            <option key={c}>{c}</option>
                          ))}
                        </select>
                      </label>
                    </div>
                    <Field
                      label="Description"
                      name="description"
                      required={false}
                    />
                    <label>
                      Polygon coordinates
                      <textarea
                        required
                        name="coordinates"
                        key={JSON.stringify(draft)}
                        defaultValue={JSON.stringify(draft)}
                      />
                    </label>
                    <button disabled={busy} className="primary">
                      {busy ? "Saving…" : "Save geofence"}
                    </button>
                  </form>
                  <hr />
                  <h2>Geofence directory</h2>
                  {fences
                    .filter((f) => !category || f.category === category)
                    .map((f) => (
                      <div className="list-row" key={f.id}>
                        <div>
                          <b>{f.name}</b>
                          <p>{f.description || "No description"}</p>
                        </div>
                        <span className="badge">{label(f.category)}</span>
                      </div>
                    ))}
                  {!fences.length && (
                    <Empty text="Create your first boundary to start monitoring." />
                  )}
                </section>
              )}
              {tab === "Vehicles" && (
                <section className="card panel">
                  <h2>Register a vehicle</h2>
                  <form onSubmit={(e) => submit(e, "/vehicles")}>
                    <div className="form-grid">
                      <Field label="Vehicle number" name="vehicle_number" />
                      <Field label="Driver name" name="driver_name" />
                      <Field
                        label="Vehicle type"
                        name="vehicle_type"
                        placeholder="truck"
                      />
                      <Field label="Phone" name="phone" type="tel" />
                    </div>
                    <button className="primary" disabled={busy}>
                      Register vehicle
                    </button>
                  </form>
                  <hr />
                  <h2>Update location</h2>
                  <p>
                    Select a vehicle, then click the map or enter its
                    coordinates.
                  </p>
                  <form
                    onSubmit={(e) =>
                      submit(e, "/vehicles/location", (d) => ({
                        ...d,
                        latitude: Number(d.latitude),
                        longitude: Number(d.longitude),
                        timestamp: new Date(d.timestamp).toISOString(),
                      }))
                    }
                  >
                    <label>
                      Vehicle
                      <select
                        required
                        name="vehicle_id"
                        value={selected}
                        onChange={(e) => setSelected(e.target.value)}
                      >
                        <option value="">Select vehicle</option>
                        {vehicles.map((v) => (
                          <option key={v.id} value={v.id}>
                            {v.vehicle_number}
                          </option>
                        ))}
                      </select>
                    </label>
                    <button type="button" onClick={() => setMode("move")}>
                      Choose on map
                    </button>
                    <div className="form-grid">
                      <label>
                        Latitude
                        <input
                          required
                          name="latitude"
                          type="number"
                          step="any"
                          min="-90"
                          max="90"
                          value={point[0]}
                          onChange={(e) => setPoint([e.target.value, point[1]])}
                        />
                      </label>
                      <label>
                        Longitude
                        <input
                          required
                          name="longitude"
                          type="number"
                          step="any"
                          min="-180"
                          max="180"
                          value={point[1]}
                          onChange={(e) => setPoint([point[0], e.target.value])}
                        />
                      </label>
                    </div>
                    <Field
                      label="Recorded at"
                      name="timestamp"
                      type="datetime-local"
                      defaultValue={new Date(
                        Date.now() - new Date().getTimezoneOffset() * 60000,
                      )
                        .toISOString()
                        .slice(0, 19)}
                      step="1"
                    />
                    <button className="primary" disabled={busy}>
                      Update vehicle location
                    </button>
                  </form>
                  <hr />
                  <Fleet vehicles={vehicles} positions={positions} />
                </section>
              )}
              {tab === "Alert rules" && (
                <section className="card panel">
                  <h2>Configure a rule</h2>
                  <p>
                    Receive one live alert per matching vehicle transition, even
                    when multiple rules match.
                  </p>
                  <form onSubmit={(e) => submit(e, "/alerts/configure")}>
                    <div className="form-grid">
                      <label>
                        Geofence
                        <select required name="geofence_id">
                          {fenceOptions}
                        </select>
                      </label>
                      <label>
                        Vehicle
                        <select name="vehicle_id">{vehicleOptions}</select>
                      </label>
                      <label>
                        Event type
                        <select name="event_type">
                          <option value="both">Entry and exit</option>
                          <option value="entry">Entry only</option>
                          <option value="exit">Exit only</option>
                        </select>
                      </label>
                    </div>
                    <button className="primary" disabled={busy}>
                      Activate rule
                    </button>
                  </form>
                  <hr />
                  <h2>Active rules</h2>
                  {rules.map((r) => (
                    <div className="list-row" key={r.alert_id}>
                      <div>
                        <b>{r.geofence_name}</b>
                        <p>
                          {r.vehicle_number || "All vehicles"} · {r.event_type}
                        </p>
                      </div>
                      <span className="badge">Active</span>
                    </div>
                  ))}
                  {!rules.length && (
                    <Empty text="Configure a rule to receive live alerts." />
                  )}
                </section>
              )}
              {tab === "Event history" && (
                <section className="card panel">
                  <h2>Movement history</h2>
                  <div className="form-grid">
                    <label>
                      Vehicle
                      <select
                        value={filters.vehicle_id}
                        onChange={(e) => {
                          setOffset(0);
                          setFilters({
                            ...filters,
                            vehicle_id: e.target.value,
                          });
                        }}
                      >
                        {vehicleOptions}
                      </select>
                    </label>
                    <label>
                      Geofence
                      <select
                        value={filters.geofence_id}
                        onChange={(e) => {
                          setOffset(0);
                          setFilters({
                            ...filters,
                            geofence_id: e.target.value,
                          });
                        }}
                      >
                        <option value="">All geofences</option>
                        {fences.map((f) => (
                          <option key={f.id} value={f.id}>
                            {f.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    {["start_date", "end_date"].map((k) => (
                      <label key={k}>
                        {label(k)}
                        <input
                          type="datetime-local"
                          value={filters[k]}
                          onChange={(e) => {
                            setOffset(0);
                            setFilters({ ...filters, [k]: e.target.value });
                          }}
                        />
                      </label>
                    ))}
                  </div>
                  <div className="table-scroll">
                    <table>
                      <thead>
                        <tr>
                          <th>Vehicle / zone</th>
                          <th>Event</th>
                          <th>Location</th>
                          <th>Recorded at</th>
                        </tr>
                      </thead>
                      <tbody>
                        {history.map((h) => (
                          <tr key={h.id}>
                            <td>
                              <b>{h.vehicle_number}</b>
                              <small>{h.geofence_name}</small>
                            </td>
                            <td>
                              <span className="badge">{h.event_type}</span>
                            </td>
                            <td>
                              {h.latitude.toFixed(4)}, {h.longitude.toFixed(4)}
                            </td>
                            <td>{new Date(h.timestamp).toLocaleString()}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  {!history.length && (
                    <Empty text="No events match these filters." />
                  )}
                  <div className="pagination">
                    <span>{total} matching events</span>
                    <button
                      disabled={!offset}
                      onClick={() => setOffset(Math.max(0, offset - 10))}
                    >
                      ← Previous
                    </button>
                    <button
                      disabled={offset + 10 >= total}
                      onClick={() => setOffset(offset + 10)}
                    >
                      Next →
                    </button>
                  </div>
                </section>
              )}
            </div>
            <section className="card feed">
              <div className="card-head">
                <div>
                  <h2>
                    Live activity <span className="live-tag">LIVE</span>
                  </h2>
                  <p>Matching alerts, as they happen</p>
                </div>
              </div>
              <div className="feed-body">
                {feed.length ? (
                  feed.map((f) => (
                    <article className="event" key={f.event_id}>
                      <div className={"event-icon " + f.event_type}>
                        {f.event_type === "entry" ? "↘" : "↗"}
                      </div>
                      <div>
                        <b>{f.vehicle.vehicle_number}</b>
                        <span className="event-time">
                          {new Date(f.timestamp).toLocaleTimeString()}
                        </span>
                        <p>
                          {f.event_type === "entry" ? "Entered" : "Exited"}{" "}
                          <strong>{f.geofence.geofence_name}</strong>
                        </p>
                        <span className="badge">
                          {label(f.geofence.category)}
                        </span>
                        <small>
                          {f.location.latitude.toFixed(4)},{" "}
                          {f.location.longitude.toFixed(4)}
                        </small>
                      </div>
                    </article>
                  ))
                ) : (
                  <Empty text="You're all caught up. Alerts appear here when a vehicle crosses a configured boundary." />
                )}
              </div>
              <div className="feed-foot">
                ●{" "}
                {connected
                  ? "Listening for fleet events"
                  : "Waiting for connection"}
              </div>
            </section>
          </div>
          <footer>
            Boundary · Geofencing & real-time alerts{" "}
            <span>All coordinates use latitude, longitude</span>
          </footer>
        </section>
      </main>
    </div>
  );
}
function Field({ label, ...props }) {
  return (
    <label>
      {label}
      <input required {...props} />
    </label>
  );
}
function Empty({ text }) {
  return (
    <div className="empty">
      <span>◎</span>
      <p>{text}</p>
    </div>
  );
}
function Fleet({ vehicles, positions }) {
  return (
    <div className="table-scroll">
      <table>
        <thead>
          <tr>
            <th>Vehicle</th>
            <th>Driver</th>
            <th>Current zone</th>
          </tr>
        </thead>
        <tbody>
          {vehicles.map((v) => {
            const p = positions.find((p) => p.vehicle_id === v.id);
            return (
              <tr key={v.id}>
                <td>
                  <b>{v.vehicle_number}</b>
                  <small>{v.vehicle_type}</small>
                </td>
                <td>{v.driver_name}</td>
                <td>
                  <span className="badge">
                    {p?.current_geofences
                      .map((g) => g.geofence_name)
                      .join(", ") ||
                      (p?.current_location
                        ? "Outside zones"
                        : "Awaiting location")}
                  </span>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      {!vehicles.length && (
        <Empty text="Your fleet starts here. Register a vehicle to see it on the map." />
      )}
    </div>
  );
}
createRoot(document.getElementById("root")).render(<App />);
