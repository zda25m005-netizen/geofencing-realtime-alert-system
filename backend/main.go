package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type object = map[string]any
type App struct {
	db        *database
	mu        sync.Mutex
	clients   map[chan []byte]bool
	clientsMu sync.Mutex
	origin    string
}
type Fence struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Coordinates [][]float64 `json:"coordinates"`
	Category    string      `json:"category"`
	Created     string      `json:"created_at"`
}
type Vehicle struct {
	ID      string `json:"id"`
	Number  string `json:"vehicle_number"`
	Driver  string `json:"driver_name"`
	Type    string `json:"vehicle_type"`
	Phone   string `json:"phone"`
	Status  string `json:"status"`
	Created string `json:"created_at"`
}
type Location struct {
	Vehicle   string   `json:"vehicle_id"`
	Lat       *float64 `json:"latitude"`
	Lng       *float64 `json:"longitude"`
	Timestamp string   `json:"timestamp"`
}
type Rule struct {
	ID            string `json:"alert_id"`
	Fence         string `json:"geofence_id"`
	Vehicle       string `json:"vehicle_id"`
	Event         string `json:"event_type"`
	Status        string `json:"status"`
	Created       string `json:"created_at"`
	FenceName     string `json:"geofence_name"`
	VehicleNumber string `json:"vehicle_number"`
}

func id(prefix string) string {
	b := make([]byte, 12)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return fmt.Sprintf("%s_%x", prefix, b)
}
func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func newApp(path string) (*App, error) {
	db, e := openDatabase(path)
	if e != nil {
		return nil, e
	}
	return &App{db: db, clients: map[chan []byte]bool{}, origin: env("CORS_ORIGIN", "http://localhost:5173")}, nil
}

func encode(v any) string { b, _ := json.Marshal(v); return string(b) }
func reply(w http.ResponseWriter, start time.Time, status int, v object) {
	v["time_ns"] = strconv.FormatInt(time.Since(start).Nanoseconds(), 10)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return errors.New("invalid JSON: " + e.Error())
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errors.New("body must contain one JSON value")
	}
	return nil
}
func (a *App) fences() ([]Fence, error) {
	rows, e := a.db.Query("SELECT data FROM fences ORDER BY rowid")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Fence{}
	for rows.Next() {
		var s string
		var f Fence
		if e = rows.Scan(&s); e != nil {
			return nil, e
		}
		if e = json.Unmarshal([]byte(s), &f); e != nil {
			return nil, e
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
func (a *App) vehicles() ([]Vehicle, error) {
	rows, e := a.db.Query("SELECT data FROM vehicles ORDER BY rowid")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Vehicle{}
	for rows.Next() {
		var s string
		var v Vehicle
		if e = rows.Scan(&s); e != nil {
			return nil, e
		}
		if e = json.Unmarshal([]byte(s), &v); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (a *App) rules() ([]Rule, error) {
	rows, e := a.db.Query("SELECT data FROM rules ORDER BY rowid")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Rule{}
	for rows.Next() {
		var s string
		var v Rule
		if e = rows.Scan(&s); e != nil {
			return nil, e
		}
		if e = json.Unmarshal([]byte(s), &v); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func validFence(f Fence) error {
	if strings.TrimSpace(f.Name) == "" {
		return errors.New("name is required")
	}
	switch f.Category {
	case "delivery_zone", "restricted_zone", "toll_zone", "customer_area":
	default:
		return errors.New("invalid category")
	}
	p := f.Coordinates
	if len(p) > 1000 {
		return errors.New("polygon must have at most 1000 points")
	}
	if len(p) < 4 {
		return errors.New("polygon needs at least 4 points including closure")
	}
	unique := map[string]bool{}
	area := 0.0
	for i, c := range p {
		if len(c) != 2 || !validPoint(c[0], c[1]) {
			return errors.New("coordinates must be [latitude, longitude] pairs in range")
		}
		unique[encode(c)] = true
		if i > 0 {
			area += p[i-1][0]*c[1] - c[0]*p[i-1][1]
		}
	}
	if p[0][0] != p[len(p)-1][0] || p[0][1] != p[len(p)-1][1] {
		return errors.New("first and last coordinates must match")
	}
	if len(unique) < 3 || math.Abs(area) < 1e-12 {
		return errors.New("polygon requires 3 distinct non-collinear points")
	}
	for i := 0; i < len(p)-1; i++ {
		for j := i + 1; j < len(p)-1; j++ {
			if j == i+1 || (i == 0 && j == len(p)-2) {
				continue
			}
			if intersects(p[i], p[i+1], p[j], p[j+1]) {
				return errors.New("polygon must not intersect itself")
			}
		}
	}
	return nil
}
func cross(a, b, c []float64) float64 { return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0]) }
func on(a, b, c []float64) bool {
	return math.Abs(cross(a, b, c)) < 1e-10 && c[0] >= math.Min(a[0], b[0])-1e-10 && c[0] <= math.Max(a[0], b[0])+1e-10 && c[1] >= math.Min(a[1], b[1])-1e-10 && c[1] <= math.Max(a[1], b[1])+1e-10
}
func intersects(a, b, c, d []float64) bool {
	return cross(a, b, c)*cross(a, b, d) < 0 && cross(c, d, a)*cross(c, d, b) < 0 || on(a, b, c) || on(a, b, d) || on(c, d, a) || on(c, d, b)
}
func validPoint(lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// Boundary points count as inside. Coordinates always remain latitude, longitude.
func inside(lat, lng float64, p [][]float64) bool {
	result := false
	point := []float64{lat, lng}
	for i := 0; i < len(p)-1; i++ {
		a, b := p[i], p[i+1]
		if on(a, b, point) {
			return true
		}
		if (a[1] > lng) != (b[1] > lng) && lat < (b[0]-a[0])*(lng-a[1])/(b[1]-a[1])+a[0] {
			result = !result
		}
	}
	return result
}
func current(lat, lng float64, fs []Fence) []object {
	out := []object{}
	for _, f := range fs {
		if inside(lat, lng, f.Coordinates) {
			out = append(out, object{"geofence_id": f.ID, "geofence_name": f.Name, "category": f.Category, "status": "inside"})
		}
	}
	return out
}
func (a *App) broadcast(b []byte) {
	a.clientsMu.Lock()
	defer a.clientsMu.Unlock()
	for ch := range a.clients {
		select {
		case ch <- b:
		default:
			delete(a.clients, ch)
			close(ch)
		}
	}
}
func (a *App) socket(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	u := websocket.Upgrader{Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		reply(w, start, status, object{"error": reason.Error()})
	}, CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == "" || r.Header.Get("Origin") == a.origin }}
	c, e := u.Upgrade(w, r, nil)
	if e != nil {
		return
	}
	defer c.Close()
	ch := make(chan []byte, 64)
	a.clientsMu.Lock()
	a.clients[ch] = true
	a.clientsMu.Unlock()
	defer func() { a.clientsMu.Lock(); delete(a.clients, ch); a.clientsMu.Unlock() }()
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.SetReadLimit(1024)
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.SetPongHandler(func(string) error { return c.SetReadDeadline(time.Now().Add(60 * time.Second)) })
		for {
			if _, _, e := c.ReadMessage(); e != nil {
				return
			}
		}
	}()
	tick := time.NewTicker(25 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-done:
			return
		case b, ok := <-ch:
			if !ok {
				return
			}
			c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if c.WriteMessage(websocket.TextMessage, b) != nil {
				return
			}
		case <-tick.C:
			c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if c.WriteMessage(websocket.PingMessage, nil) != nil {
				return
			}
		}
	}
}
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Access-Control-Allow-Origin", a.origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	fail := func(code int, msg string) { reply(w, start, code, object{"error": msg}) }
	if r.Method == "OPTIONS" {
		reply(w, start, 200, object{})
		return
	}
	if r.URL.Path == "/ws/alerts" && r.Method == "GET" {
		a.socket(w, r)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock() // Serialize writes and containment comparisons, preserving per-vehicle order.
	internal := func(e error) { log.Print(e); fail(500, "database operation failed") }
	switch {
	case r.URL.Path == "/health" && r.Method == "GET":
		reply(w, start, 200, object{"status": "ok"})
	case r.URL.Path == "/geofences" && r.Method == "POST":
		var f Fence
		if e := decode(w, r, &f); e != nil {
			fail(400, e.Error())
			return
		}
		if e := validFence(f); e != nil {
			fail(400, e.Error())
			return
		}
		f.ID = id("geo")
		f.Created = now()
		if _, e := a.db.Exec("INSERT INTO fences(id,data) VALUES(?,?)", f.ID, encode(f)); e != nil {
			internal(e)
			return
		}
		reply(w, start, 201, object{"id": f.ID, "name": f.Name, "status": "active"})
	case r.URL.Path == "/geofences" && r.Method == "GET":
		fs, e := a.fences()
		if e != nil {
			internal(e)
			return
		}
		out := []Fence{}
		for _, f := range fs {
			if c := r.URL.Query().Get("category"); c == "" || c == f.Category {
				out = append(out, f)
			}
		}
		reply(w, start, 200, object{"geofences": out})
	case r.URL.Path == "/vehicles" && r.Method == "POST":
		var v Vehicle
		if e := decode(w, r, &v); e != nil {
			fail(400, e.Error())
			return
		}
		if strings.TrimSpace(v.Number) == "" || strings.TrimSpace(v.Driver) == "" || strings.TrimSpace(v.Type) == "" || strings.TrimSpace(v.Phone) == "" {
			fail(400, "vehicle_number, driver_name, vehicle_type and phone are required")
			return
		}
		v.ID = id("veh")
		v.Created = now()
		v.Status = "active"
		_, e := a.db.Exec("INSERT INTO vehicles(id,number,data) VALUES(?,?,?)", v.ID, v.Number, encode(v))
		if e != nil {
			if strings.Contains(e.Error(), "UNIQUE") || strings.Contains(e.Error(), "23505") {
				fail(409, "vehicle_number already registered")
			} else {
				internal(e)
			}
			return
		}
		reply(w, start, 201, object{"id": v.ID, "vehicle_number": v.Number, "status": v.Status})
	case r.URL.Path == "/vehicles" && r.Method == "GET":
		vs, e := a.vehicles()
		if e != nil {
			internal(e)
			return
		}
		reply(w, start, 200, object{"vehicles": vs})
	case r.URL.Path == "/vehicles/location" && r.Method == "POST":
		var l Location
		if e := decode(w, r, &l); e != nil {
			fail(400, e.Error())
			return
		}
		ts, e := time.Parse(time.RFC3339Nano, l.Timestamp)
		if l.Lat == nil || l.Lng == nil || !validPoint(*l.Lat, *l.Lng) || e != nil {
			fail(400, "latitude, longitude and a valid RFC3339 timestamp are required")
			return
		}
		l.Timestamp = ts.UTC().Format(time.RFC3339Nano)
		var raw string
		if e = a.db.QueryRow("SELECT data FROM vehicles WHERE id=?", l.Vehicle).Scan(&raw); e != nil {
			if e == sql.ErrNoRows {
				fail(404, "vehicle not found")
			} else {
				internal(e)
			}
			return
		}
		var v Vehicle
		json.Unmarshal([]byte(raw), &v)
		fs, e := a.fences()
		if e != nil {
			internal(e)
			return
		}
		rs, e := a.rules()
		if e != nil {
			internal(e)
			return
		}
		var oldLat, oldLng float64
		var oldTime string
		e = a.db.QueryRow("SELECT lat,lng,timestamp FROM locations WHERE vehicle=? ORDER BY id DESC LIMIT 1", l.Vehicle).Scan(&oldLat, &oldLng, &oldTime)
		hadOld := e == nil
		if e != nil && e != sql.ErrNoRows {
			internal(e)
			return
		}
		if hadOld {
			ot, _ := time.Parse(time.RFC3339Nano, oldTime)
			if !ts.After(ot) {
				fail(409, "timestamp must be newer than the previous location")
				return
			}
		}
		tx, e := a.db.Begin()
		if e != nil {
			internal(e)
			return
		}
		defer tx.Rollback()
		if _, e = tx.Exec("INSERT INTO locations(vehicle,lat,lng,timestamp) VALUES(?,?,?,?)", l.Vehicle, *l.Lat, *l.Lng, l.Timestamp); e != nil {
			internal(e)
			return
		}
		messages := [][]byte{}
		for _, f := range fs {
			was := hadOld && inside(oldLat, oldLng, f.Coordinates)
			is := inside(*l.Lat, *l.Lng, f.Coordinates)
			if was == is {
				continue
			}
			event := "entry"
			if !is {
				event = "exit"
			}
			eid := id("evt")
			history := object{"id": eid, "vehicle_id": v.ID, "vehicle_number": v.Number, "geofence_id": f.ID, "geofence_name": f.Name, "event_type": event, "latitude": *l.Lat, "longitude": *l.Lng, "timestamp": l.Timestamp}
			matched := false
			for _, rule := range rs {
				if rule.Fence == f.ID && (rule.Vehicle == "" || rule.Vehicle == v.ID) && (rule.Event == "both" || rule.Event == event) {
					matched = true
				}
			}
			alert := ""
			if matched {
				msg := object{"event_id": eid, "event_type": event, "timestamp": l.Timestamp, "vehicle": object{"vehicle_id": v.ID, "vehicle_number": v.Number, "driver_name": v.Driver}, "geofence": object{"geofence_id": f.ID, "geofence_name": f.Name, "category": f.Category}, "location": object{"latitude": *l.Lat, "longitude": *l.Lng}}
				alert = encode(msg)
				messages = append(messages, []byte(alert))
			}
			if _, e = tx.Exec("INSERT INTO events(id,vehicle,fence,timestamp,data,alert) VALUES(?,?,?,?,?,?)", eid, v.ID, f.ID, l.Timestamp, encode(history), alert); e != nil {
				internal(e)
				return
			}
		}
		if e = tx.Commit(); e != nil {
			internal(e)
			return
		}
		for _, msg := range messages {
			a.broadcast(msg)
		}
		reply(w, start, 200, object{"vehicle_id": v.ID, "location_updated": true, "current_geofences": current(*l.Lat, *l.Lng, fs)})
	case strings.HasPrefix(r.URL.Path, "/vehicles/location/") && r.Method == "GET":
		vid := strings.TrimPrefix(r.URL.Path, "/vehicles/location/")
		var raw string
		e := a.db.QueryRow("SELECT data FROM vehicles WHERE id=?", vid).Scan(&raw)
		if e != nil {
			if e == sql.ErrNoRows {
				fail(404, "vehicle not found")
			} else {
				internal(e)
			}
			return
		}
		var v Vehicle
		json.Unmarshal([]byte(raw), &v)
		var lat, lng float64
		var ts string
		e = a.db.QueryRow("SELECT lat,lng,timestamp FROM locations WHERE vehicle=? ORDER BY id DESC LIMIT 1", vid).Scan(&lat, &lng, &ts)
		if e == sql.ErrNoRows {
			reply(w, start, 200, object{"vehicle_id": vid, "vehicle_number": v.Number, "current_location": nil, "current_geofences": []object{}})
			return
		}
		if e != nil {
			internal(e)
			return
		}
		fs, e := a.fences()
		if e != nil {
			internal(e)
			return
		}
		reply(w, start, 200, object{"vehicle_id": vid, "vehicle_number": v.Number, "current_location": object{"latitude": lat, "longitude": lng, "timestamp": ts}, "current_geofences": current(lat, lng, fs)})
	case r.URL.Path == "/alerts/configure" && r.Method == "POST":
		var rule Rule
		if e := decode(w, r, &rule); e != nil {
			fail(400, e.Error())
			return
		}
		if rule.Event != "entry" && rule.Event != "exit" && rule.Event != "both" {
			fail(400, "event_type must be entry, exit or both")
			return
		}
		var raw string
		e := a.db.QueryRow("SELECT data FROM fences WHERE id=?", rule.Fence).Scan(&raw)
		if e != nil {
			fail(400, "unknown geofence_id")
			return
		}
		var f Fence
		json.Unmarshal([]byte(raw), &f)
		rule.FenceName = f.Name
		if rule.Vehicle != "" {
			if e = a.db.QueryRow("SELECT data FROM vehicles WHERE id=?", rule.Vehicle).Scan(&raw); e != nil {
				fail(400, "unknown vehicle_id")
				return
			}
			var v Vehicle
			json.Unmarshal([]byte(raw), &v)
			rule.VehicleNumber = v.Number
		}
		rule.ID = id("alert")
		rule.Status = "active"
		rule.Created = now()
		if _, e = a.db.Exec("INSERT INTO rules(id,fence,vehicle,event,data) VALUES(?,?,?,?,?)", rule.ID, rule.Fence, rule.Vehicle, rule.Event, encode(rule)); e != nil {
			internal(e)
			return
		}
		reply(w, start, 201, object{"alert_id": rule.ID, "geofence_id": rule.Fence, "vehicle_id": rule.Vehicle, "event_type": rule.Event, "status": rule.Status})
	case r.URL.Path == "/alerts" && r.Method == "GET":
		rs, e := a.rules()
		if e != nil {
			internal(e)
			return
		}
		out := []Rule{}
		q := r.URL.Query()
		for _, v := range rs {
			if (q.Get("vehicle_id") == "" || v.Vehicle == q.Get("vehicle_id")) && (q.Get("geofence_id") == "" || v.Fence == q.Get("geofence_id")) {
				out = append(out, v)
			}
		}
		reply(w, start, 200, object{"alerts": out})
	case r.URL.Path == "/violations/history" && r.Method == "GET":
		q := r.URL.Query()
		limit, offset := 50, 0
		for _, k := range []string{"limit", "offset"} {
			if q.Get(k) != "" {
				n, e := strconv.Atoi(q.Get(k))
				if e != nil || n < 0 || (k == "limit" && (n < 1 || n > 500)) {
					fail(400, "limit must be 1–500; offset must be nonnegative")
					return
				}
				if k == "limit" {
					limit = n
				} else {
					offset = n
				}
			}
		}
		var startDate, endDate time.Time
		for _, k := range []string{"start_date", "end_date"} {
			if q.Get(k) != "" {
				t, e := time.Parse(time.RFC3339Nano, q.Get(k))
				if e != nil {
					fail(400, k+" must be RFC3339")
					return
				}
				if k == "start_date" {
					startDate = t
				} else {
					endDate = t
				}
			}
		}
		if !startDate.IsZero() && !endDate.IsZero() && startDate.After(endDate) {
			fail(400, "start_date must precede end_date")
			return
		}
		rows, e := a.db.Query("SELECT data FROM events ORDER BY rowid DESC")
		if e != nil {
			internal(e)
			return
		}
		out := []object{}
		total := 0
		for rows.Next() {
			var raw string
			if e = rows.Scan(&raw); e != nil {
				break
			}
			var v object
			if e = json.Unmarshal([]byte(raw), &v); e != nil {
				break
			}
			t, _ := time.Parse(time.RFC3339Nano, v["timestamp"].(string))
			if (q.Get("vehicle_id") != "" && v["vehicle_id"] != q.Get("vehicle_id")) || (q.Get("geofence_id") != "" && v["geofence_id"] != q.Get("geofence_id")) || (!startDate.IsZero() && t.Before(startDate)) || (!endDate.IsZero() && t.After(endDate)) {
				continue
			}
			if total >= offset && len(out) < limit {
				out = append(out, v)
			}
			total++
		}
		rowErr := rows.Err()
		rows.Close()
		if e != nil {
			internal(e)
			return
		}
		if rowErr != nil {
			internal(rowErr)
			return
		}
		reply(w, start, 200, object{"violations": out, "total_count": total})
	default:
		fail(404, "endpoint or method not found")
	}
}
func main() {
	path := env("DATABASE_URL", env("DATABASE_PATH", "./data/geofence.db"))
	if !isPostgres(path) {
		if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
			log.Fatal(e)
		}
	}
	a, e := newApp(path)
	if e != nil {
		log.Fatal(e)
	}
	defer a.db.Close()
	s := &http.Server{Addr: ":" + env("PORT", "8080"), Handler: a, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("API listening on %s", s.Addr)
	log.Fatal(s.ListenAndServe())
}
