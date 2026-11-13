package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPolygon(t *testing.T) {
	p := [][]float64{{0, 0}, {0, 2}, {2, 2}, {2, 0}, {0, 0}}
	for _, c := range []struct {
		lat, lng float64
		want     bool
	}{{1, 1, true}, {0, 1, true}, {0, 0, true}, {3, 1, false}, {1, 3, false}} {
		if got := inside(c.lat, c.lng, p); got != c.want {
			t.Fatalf("inside(%v,%v)=%v", c.lat, c.lng, got)
		}
	}
	for _, poly := range [][][]float64{{{0, 0}, {1, 1}, {2, 2}, {0, 0}}, {{0, 0}, {1, 1}, {0, 1}, {1, 0}, {0, 0}}, {{0, 0}, {0, 1}, {1, 1}}} {
		if validFence(Fence{Name: "x", Category: "restricted_zone", Coordinates: poly}) == nil {
			t.Fatal("accepted invalid polygon")
		}
	}
}
func TestSystem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	a, e := newApp(path)
	if e != nil {
		t.Fatal(e)
	}
	s := httptest.NewServer(a)
	defer s.Close()
	defer a.db.Close()
	call := func(method, path string, body any, status int) object {
		t.Helper()
		b, _ := json.Marshal(body)
		r, e := http.NewRequest(method, s.URL+path, bytes.NewReader(b))
		if e != nil {
			t.Fatal(e)
		}
		r.Header.Set("Content-Type", "application/json")
		res, e := http.DefaultClient.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		var out object
		if e = json.NewDecoder(res.Body).Decode(&out); e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != status {
			t.Fatalf("%s %s: status %d, want %d: %v", method, path, res.StatusCode, status, out)
		}
		if _, ok := out["time_ns"].(string); !ok {
			t.Fatal("missing string time_ns")
		}
		return out
	}
	call("POST", "/geofences", object{"name": "bad", "category": "restricted_zone", "coordinates": [][]float64{{1, 2}}}, 400)
	f := call("POST", "/geofences", object{"name": "Depot", "description": "Test", "category": "restricted_zone", "coordinates": [][]float64{{0, 0}, {0, 2}, {2, 2}, {2, 0}, {0, 0}}}, 201)["id"].(string)
	v := call("POST", "/vehicles", object{"vehicle_number": "TEST-1", "driver_name": "A", "vehicle_type": "truck", "phone": "123"}, 201)["id"].(string)
	call("POST", "/vehicles", object{"vehicle_number": "TEST-1", "driver_name": "B", "vehicle_type": "car", "phone": "123"}, 409)
	call("POST", "/alerts/configure", object{"geofence_id": f, "event_type": "both"}, 201)
	call("POST", "/alerts/configure", object{"geofence_id": f, "vehicle_id": v, "event_type": "entry"}, 201)
	for _, path := range []string{"/geofences?category=restricted_zone", "/vehicles", "/alerts?geofence_id=" + f, "/vehicles/location/" + v} {
		call("GET", path, nil, 200)
	}
	c1, _, e := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+"/ws/alerts", nil)
	if e != nil {
		t.Fatal(e)
	}
	defer c1.Close()
	c2, _, e := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+"/ws/alerts", nil)
	if e != nil {
		t.Fatal(e)
	}
	defer c2.Close()
	base := time.Now().UTC()
	update := func(lat, lng float64, n int) {
		call("POST", "/vehicles/location", object{"vehicle_id": v, "latitude": lat, "longitude": lng, "timestamp": base.Add(time.Duration(n) * time.Second).Format(time.RFC3339Nano)}, 200)
	}
	update(3, 3, 0)
	update(1, 1, 1)
	update(1.1, 1.1, 2)
	update(3, 3, 3)
	for _, c := range []*websocket.Conn{c1, c2} {
		for _, want := range []string{"entry", "exit"} {
			c.SetReadDeadline(time.Now().Add(3 * time.Second))
			var msg object
			if e := c.ReadJSON(&msg); e != nil {
				t.Fatal(e)
			}
			if msg["event_type"] != want {
				t.Fatalf("got %v, want %s", msg, want)
			}
		}
	}
	call("POST", "/vehicles/location", object{"vehicle_id": v, "latitude": 0, "longitude": 0, "timestamp": base.Format(time.RFC3339Nano)}, 409)
	call("POST", "/vehicles/location", object{"vehicle_id": v, "longitude": 0, "timestamp": base.Format(time.RFC3339Nano)}, 400)
	h := call("GET", "/violations/history?vehicle_id="+v+"&limit=1&offset=1", nil, 200)
	if h["total_count"] != float64(2) || len(h["violations"].([]any)) != 1 {
		t.Fatalf("wrong pagination %v", h)
	}
	call("GET", "/violations/history?start_date=no", nil, 400)
	call("GET", "/missing", nil, 404)
	v2 := call("POST", "/vehicles", object{"vehicle_number": "TEST-2", "driver_name": "B", "vehicle_type": "van", "phone": "123"}, 201)["id"].(string)
	call("POST", "/vehicles/location", object{"vehicle_id": v2, "latitude": 1, "longitude": 1, "timestamp": base.Format(time.RFC3339Nano)}, 200)
	a.db.Close()
	reopened, e := newApp(path)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.db.Close()
	var count int
	if e = reopened.db.QueryRow("SELECT count(*) FROM events").Scan(&count); e != nil || count != 3 {
		t.Fatalf("persistence failed: count=%d err=%v", count, e)
	}
}
