package store

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const metaAsOf = "as_of"

// SetMeta stores a key/value pair in the meta table.
func (s *Store) SetMeta(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("meta key is empty")
	}
	_, err := s.db.Exec(`INSERT INTO meta(key,value) VALUES(?,?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return wrap("set meta", err)
}

// GetMeta returns a meta value. ok=false when the key is absent.
func (s *Store) GetMeta(key string) (value string, ok bool, err error) {
	err = s.db.QueryRow(`SELECT value FROM meta WHERE key=?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, wrap("get meta", err)
	}
	return value, true, nil
}

// SetAsOf persists the effective incident clock used by ask/verify.
func (s *Store) SetAsOf(t time.Time) error {
	if t.IsZero() {
		return fmt.Errorf("as_of must be non-zero")
	}
	return s.SetMeta(metaAsOf, fmtTime(t))
}

// ClearAsOf removes a persisted incident clock so readers use wall time
// (live ingest must not freeze ages/R1 windows to the ingest instant).
func (s *Store) ClearAsOf() error {
	_, err := s.db.Exec(`DELETE FROM meta WHERE key=?`, metaAsOf)
	return wrap("clear as_of", err)
}

// AsOf returns the persisted incident clock, if any.
func (s *Store) AsOf() (time.Time, bool, error) {
	v, ok, err := s.GetMeta(metaAsOf)
	if err != nil || !ok {
		return time.Time{}, ok, err
	}
	t, ok := parseTimeOK(v)
	if !ok {
		return time.Time{}, false, fmt.Errorf("corrupt as_of meta value %q", v)
	}
	return t, true, nil
}

// FindAliasCollisions reports name/alias keys that resolve to more than one
// service (case-insensitive). Each entry is "key => id1, id2".
func (s *Store) FindAliasCollisions() ([]string, error) {
	svcs, err := s.ListServices()
	if err != nil {
		return nil, err
	}
	type hit struct {
		ids []string
	}
	by := map[string]*hit{}
	add := func(key, id string) {
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			return
		}
		h := by[key]
		if h == nil {
			h = &hit{}
			by[key] = h
		}
		for _, existing := range h.ids {
			if existing == id {
				return
			}
		}
		h.ids = append(h.ids, id)
	}
	for _, svc := range svcs {
		add(svc.ID, svc.ID)
		add(svc.Name, svc.ID)
		for _, a := range svc.Aliases {
			add(a, svc.ID)
		}
	}
	var out []string
	for key, h := range by {
		if len(h.ids) < 2 {
			continue
		}
		sort.Strings(h.ids)
		out = append(out, fmt.Sprintf("%q matches %s", key, strings.Join(h.ids, ", ")))
	}
	sort.Strings(out)
	return out, nil
}
