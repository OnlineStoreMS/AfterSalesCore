package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xuri/excelize/v2"
)

var png1x1 = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
	0x0A, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}

func TestUpgradeExportImageURL(t *testing.T) {
	got := upgradeExportImageURL("//p3-aio.ecombdimg.com/img/x~48x0_q90_b0.jpg")
	want := "https://p3-aio.ecombdimg.com/img/x~240x240_q90_b0.jpg"
	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestAddExportPictureEmbedsPNG(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	if err := addExportPicture(f, "Sheet1", "A2", png1x1, ".png"); err != nil {
		t.Fatal(err)
	}
	pics, err := f.GetPictures("Sheet1", "A2")
	if err != nil {
		t.Fatal(err)
	}
	if len(pics) != 1 || len(pics[0].File) == 0 {
		t.Fatalf("expected embedded picture, got %+v", pics)
	}
}

func TestFetchExportImageUsesUpgradedURL(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png1x1)
	}))
	defer srv.Close()
	_, ext, err := fetchExportImage(srv.Client(), srv.URL+"/img/x~48x0_q90_b0.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if ext != ".png" {
		t.Fatalf("ext %s", ext)
	}
	if gotPath != "/img/x~240x240_q90_b0.jpg" {
		t.Fatalf("path %s", gotPath)
	}
}
