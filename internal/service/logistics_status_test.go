package service

<<<<<<< HEAD
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
=======
import "testing"

func TestParseLogisticsTracksCollapsesDuplicatedDetail(t *testing.T) {
	msg := "您的快件已在代收点取出签收，签收代收点：菜鸟-杭州萧山利二花苑店，如遇问题请联系代收点电话【18613951558】。网点电话：0571-28150483，投诉电话：0571-28150483。感谢使用中通快递，期待再次为您服务！"
	raw := `[{"date":"09/16 23:01:42","title":"已签收","detail":"` + msg + msg + `","text":"09/16 23:01:42 已签收 ` + msg + msg + `"}]`
	tracks := ParseLogisticsTracks(raw)
	if len(tracks) != 1 {
		t.Fatalf("got %d tracks", len(tracks))
	}
	if tracks[0].Date != "09/16 23:01:42" {
		t.Fatalf("date=%q", tracks[0].Date)
	}
	if tracks[0].Title != "已签收" {
		t.Fatalf("title=%q", tracks[0].Title)
	}
	if tracks[0].Detail != msg {
		t.Fatalf("detail not collapsed:\n%s", tracks[0].Detail)
	}
	if tracks[0].Text != "09/16 23:01:42 已签收 "+msg {
		t.Fatalf("text=%q", tracks[0].Text)
	}
}

func TestCollapseDuplicatedTextTruncatedSecondCopy(t *testing.T) {
	a := "快件已送达【菜鸟-杭州萧山利二花苑店，电话：18613951558】，取件地址：杭州萧山利二花苑店，请及时取件。"
	b := a[:len(a)-len("请及时取件。")]
	got := collapseDuplicatedText(a + b)
	if got != a {
		t.Fatalf("got %q", got)
	}
}

func TestParseLogisticsTracksKeepsFive(t *testing.T) {
	raw := `[
		{"date":"09/16 23:01:42","title":"已签收","detail":"签收完成"},
		{"date":"09/12 11:37:01","title":"待取件","detail":"请及时取件"},
		{"date":"09/12 07:23:57","title":"派件中","detail":"正在派件"},
		{"date":"09/12 02:46:51","title":"运输中","detail":"已到达"},
		{"date":"09/11 17:40:37","title":"","detail":"【杭州市】快件已发往 萧山亚运村"},
		{"date":"09/11 12:00:00","title":"已发货","detail":"应被截断"}
	]`
	tracks := ParseLogisticsTracks(raw)
	if len(tracks) != 5 {
		t.Fatalf("got %d", len(tracks))
	}
	if tracks[4].Detail != "【杭州市】快件已发往 萧山亚运村" {
		t.Fatalf("last detail=%q", tracks[4].Detail)
>>>>>>> da7b61f (update)
	}
}
