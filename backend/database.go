package main

import (
	"context"
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
	"strconv"
	"strings"
	"time"
)

// Translate only repository-owned parameterized SQL, never caller-provided SQL.
type database struct {
	*sql.DB
	postgres bool
}
type transaction struct {
	*sql.Tx
	db *database
}

func isPostgres(source string) bool {
	return strings.HasPrefix(source, "postgres://") || strings.HasPrefix(source, "postgresql://")
}
func (d *database) query(q string) string {
	if !d.postgres {
		return q
	}
	q = strings.ReplaceAll(q, "rowid", "sequence")
	var out strings.Builder
	parameter := 0
	for _, c := range q {
		if c == '?' {
			parameter++
			out.WriteByte('$')
			out.WriteString(strconv.Itoa(parameter))
		} else {
			out.WriteRune(c)
		}
	}
	return out.String()
}
func (d *database) Exec(q string, args ...any) (sql.Result, error) {
	return d.DB.Exec(d.query(q), args...)
}
func (d *database) Query(q string, args ...any) (*sql.Rows, error) {
	return d.DB.Query(d.query(q), args...)
}
func (d *database) QueryRow(q string, args ...any) *sql.Row {
	return d.DB.QueryRow(d.query(q), args...)
}
func (d *database) Begin() (*transaction, error) {
	t, e := d.DB.Begin()
	if e != nil {
		return nil, e
	}
	return &transaction{Tx: t, db: d}, nil
}
func (t *transaction) Exec(q string, args ...any) (sql.Result, error) {
	return t.Tx.Exec(t.db.query(q), args...)
}
func openDatabase(source string) (*database, error) {
	postgres := isPostgres(source)
	driver := "sqlite"
	if postgres {
		driver = "pgx"
	}
	raw, e := sql.Open(driver, source)
	if e != nil {
		return nil, e
	}
	d := &database{DB: raw, postgres: postgres}
	d.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if e = d.PingContext(ctx); e != nil {
		d.Close()
		return nil, e
	}
	schema := `PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;
 CREATE TABLE IF NOT EXISTS fences(id TEXT PRIMARY KEY, data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS vehicles(id TEXT PRIMARY KEY, number TEXT UNIQUE NOT NULL, data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS locations(id INTEGER PRIMARY KEY, vehicle TEXT NOT NULL REFERENCES vehicles(id), lat REAL, lng REAL, timestamp TEXT);
 CREATE INDEX IF NOT EXISTS location_vehicle ON locations(vehicle,id DESC);
 CREATE TABLE IF NOT EXISTS rules(id TEXT PRIMARY KEY, fence TEXT REFERENCES fences(id), vehicle TEXT, event TEXT, data TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS events(id TEXT PRIMARY KEY, vehicle TEXT, fence TEXT, timestamp TEXT, data TEXT NOT NULL, alert TEXT);
 CREATE INDEX IF NOT EXISTS event_filter ON events(vehicle,fence,timestamp DESC);
 CREATE INDEX IF NOT EXISTS event_date ON events(timestamp DESC);`
	if postgres {
		schema = `
 CREATE TABLE IF NOT EXISTS fences(id TEXT PRIMARY KEY, data TEXT NOT NULL, sequence BIGSERIAL UNIQUE);
 CREATE TABLE IF NOT EXISTS vehicles(id TEXT PRIMARY KEY, number TEXT UNIQUE NOT NULL, data TEXT NOT NULL, sequence BIGSERIAL UNIQUE);
 CREATE TABLE IF NOT EXISTS locations(id BIGSERIAL PRIMARY KEY, vehicle TEXT NOT NULL REFERENCES vehicles(id), lat DOUBLE PRECISION, lng DOUBLE PRECISION, timestamp TEXT);
 CREATE INDEX IF NOT EXISTS location_vehicle ON locations(vehicle,id DESC);
 CREATE TABLE IF NOT EXISTS rules(id TEXT PRIMARY KEY, fence TEXT REFERENCES fences(id), vehicle TEXT, event TEXT, data TEXT NOT NULL, sequence BIGSERIAL UNIQUE);
 CREATE TABLE IF NOT EXISTS events(id TEXT PRIMARY KEY, vehicle TEXT, fence TEXT, timestamp TEXT, data TEXT NOT NULL, alert TEXT, sequence BIGSERIAL UNIQUE);
 CREATE INDEX IF NOT EXISTS event_filter ON events(vehicle,fence,timestamp DESC);
 CREATE INDEX IF NOT EXISTS event_date ON events(timestamp DESC);`
	}
	for _, statement := range strings.Split(schema, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, e = d.Exec(statement); e != nil {
			d.Close()
			return nil, e
		}
	}
	return d, nil
}
