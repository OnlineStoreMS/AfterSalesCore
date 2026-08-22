package plugindebug

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreSaveListGet(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	name, err := s.Save(&Record{
		RunID:   "sync-abc",
		Kind:    "sync",
		OK:      false,
		Error:   "未进入售后工作台",
		Version: "0.1.0-diag",
		ShopID:  2,
		ShopName: "觅选美妆",
		Events: []Event{
			{Ms: 0, Level: "info", Step: "start"},
			{Ms: 12, Level: "error", Step: "collect", Data: map[string]any{"url": "https://fxg.jinritemai.com/ffa/x"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(name) != ".json" {
		t.Fatalf("name %s", name)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ShopID != 2 || list[0].OK {
		t.Fatalf("list %+v", list)
	}
	got, err := s.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	if got.Error != "未进入售后工作台" || len(got.Events) != 2 {
		t.Fatalf("get %+v", got)
	}
	if _, err := s.Get("../etc/passwd"); err != ErrBadName {
		t.Fatalf("expected bad name, got %v", err)
	}
	if _, err := s.Get("missing.json"); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	_ = os.RemoveAll(dir)
}

func TestStorePrune(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.keep = 3
	for i := 0; i < 6; i++ {
		if _, err := s.Save(&Record{RunID: "r", Kind: "sync", ShopID: 1}); err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("keep 3, got %d", len(list))
	}
}
