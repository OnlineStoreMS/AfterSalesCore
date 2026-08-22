package plugindebug

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultKeep   = 80
	maxBodyBytes  = 2 << 20
	filePerm      = 0o600
	dirPerm       = 0o700
)

var nameRe = regexp.MustCompile(`^[A-Za-z0-9._-]+\.json$`)

var ErrNotFound = errors.New("debug log not found")
var ErrBadName = errors.New("invalid debug log name")

type Event struct {
	Ms    int64       `json:"ms"`
	At    string      `json:"at"`
	Level string      `json:"level"`
	Step  string      `json:"step"`
	Data  interface{} `json:"data,omitempty"`
}

type Record struct {
	RunID      string                 `json:"runId"`
	Kind       string                 `json:"kind"`
	OK         bool                   `json:"ok"`
	Error      string                 `json:"error,omitempty"`
	DurationMs int64                  `json:"durationMs"`
	Version    string                 `json:"version"`
	ShopID     uint64                 `json:"shopId"`
	ShopName   string                 `json:"shopName"`
	ReceivedAt string                 `json:"receivedAt"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
	Events     []Event                `json:"events"`
}

type Item struct {
	Name       string `json:"name"`
	RunID      string `json:"runId"`
	Kind       string `json:"kind"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"durationMs"`
	Version    string `json:"version"`
	ShopID     uint64 `json:"shopId"`
	ShopName   string `json:"shopName"`
	ReceivedAt string `json:"receivedAt"`
	Size       int64  `json:"size"`
}

type Store struct {
	dir  string
	keep int
	mu   sync.Mutex
}

func NewStore(dir string) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		dir = filepath.Join(os.TempDir(), "aftersales-plugin-debug")
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, err
	}
	return &Store{dir: dir, keep: defaultKeep}, nil
}

func (s *Store) Dir() string { return s.dir }

func (s *Store) Save(rec *Record) (string, error) {
	if rec == nil {
		return "", errors.New("empty debug log")
	}
	if rec.ReceivedAt == "" {
		rec.ReceivedAt = time.Now().Format(time.RFC3339)
	}
	if rec.Kind == "" {
		rec.Kind = "sync"
	}
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", err
	}
	if len(raw) > maxBodyBytes {
		rec.Events = rec.Events[:0]
		rec.Meta = map[string]interface{}{"truncated": true, "reason": "payload too large"}
		raw, err = json.MarshalIndent(rec, "", "  ")
		if err != nil {
			return "", err
		}
	}
	name := filename(rec)
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, name)
	if err := os.WriteFile(path, raw, filePerm); err != nil {
		return "", err
	}
	_ = s.pruneLocked()
	return name, nil
}

func (s *Store) List() ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ents, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	type fileInfo struct {
		name string
		mod  time.Time
		size int64
	}
	files := make([]fileInfo, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() || !nameRe.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: e.Name(), mod: info.ModTime(), size: info.Size()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	out := make([]Item, 0, len(files))
	for _, f := range files {
		item := Item{Name: f.name, Size: f.size}
		raw, err := os.ReadFile(filepath.Join(s.dir, f.name))
		if err == nil {
			var rec Record
			if json.Unmarshal(raw, &rec) == nil {
				item.RunID = rec.RunID
				item.Kind = rec.Kind
				item.OK = rec.OK
				item.Error = rec.Error
				item.DurationMs = rec.DurationMs
				item.Version = rec.Version
				item.ShopID = rec.ShopID
				item.ShopName = rec.ShopName
				item.ReceivedAt = rec.ReceivedAt
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Store) Get(name string) (*Record, error) {
	if !nameRe.MatchString(name) {
		return nil, ErrBadName
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *Store) pruneLocked() error {
	ents, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}
	type fileInfo struct {
		name string
		mod  time.Time
	}
	files := make([]fileInfo, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() || !nameRe.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: e.Name(), mod: info.ModTime()})
	}
	if len(files) <= s.keep {
		return nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.After(files[j].mod) })
	for _, f := range files[s.keep:] {
		_ = os.Remove(filepath.Join(s.dir, f.name))
	}
	return nil
}

func filename(rec *Record) string {
	stamp := time.Now().Format("20060102T150405")
	nano := strconv.FormatInt(time.Now().UnixNano()%1e9, 10)
	kind := sanitize(rec.Kind)
	run := sanitize(rec.RunID)
	if len(run) > 24 {
		run = run[:24]
	}
	if run == "" {
		run = "run"
	}
	return stamp + nano + "_shop" + strconv.FormatUint(rec.ShopID, 10) + "_" + kind + "_" + run + ".json"
}

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "x"
	}
	return out
}
