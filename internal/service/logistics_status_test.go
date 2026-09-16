package service

import (
	"encoding/json"
	"testing"
)

func TestParseLogisticsTracksDedupesSignedCopies(t *testing.T) {
	raw := `[
		{"date":"09/16 23:01:42","title":"已签收","detail":"09/16 23:01:42 已签收 您的快件已在代收点取出签收","text":"09/16 23:01:42 已签收 09/16 23:01:42 已签收 您的快件已在代收点取出签收"},
		{"date":"09/16 23:01:42","title":"已签收","detail":"09/16 23:01:42 已签收 您的快件已在代收点取出签收","text":"09/16 23:01:42 已签收 09/16 23:01:42 已签收 您的快件已在代收点取出签收"},
		{"date":"09/16 23:01:42","title":"已签收","detail":"09/16 23:01:42 已签收 您的快件已在代收点取出签收","text":"09/16 23:01:42 已签收 09/16 23:01:42 已签收 您的快件已在代收点取出签收"},
		{"date":"09/16 23:01:42","title":"已签收","detail":"您的快件已在代收点取出签收","text":"09/16 23:01:42 已签收 您的快件已在代收点取出签收"},
		{"date":"09/12 11:37:01","title":"待取件","detail":"09/12 11:37:01 待取件 快件已送达","text":"09/12 11:37:01 待取件 09/12 11:37:01 待取件 快件已送达"}
	]`
	tracks := ParseLogisticsTracks(raw)
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2: %+v", len(tracks), tracks)
	}
	if tracks[0].Title != "已签收" || tracks[0].Date != "09/16 23:01:42" {
		t.Fatalf("first track %+v", tracks[0])
	}
	if tracks[0].Detail != "您的快件已在代收点取出签收" {
		t.Fatalf("signed detail %q", tracks[0].Detail)
	}
	if tracks[1].Title != "待取件" || tracks[1].Detail != "快件已送达" {
		t.Fatalf("pickup track %+v", tracks[1])
	}
}

func TestLimitLogisticsTracksJSONRewritesCleanCopy(t *testing.T) {
	raw := `[{"date":"09/16 23:01:42","title":"已签收","detail":"09/16 23:01:42 已签收 已取出","text":"x"},{"date":"09/16 23:01:42","title":"已签收","detail":"已取出","text":"y"}]`
	got := LimitLogisticsTracksJSON(raw)
	var tracks []LogisticsTrack
	if err := json.Unmarshal([]byte(got), &tracks); err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].Detail != "已取出" {
		t.Fatalf("got %+v", tracks)
	}
}
