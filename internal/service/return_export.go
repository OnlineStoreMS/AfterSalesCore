package service

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"aftersalescore/internal/dto"

	"github.com/xuri/excelize/v2"
	"golang.org/x/image/webp"
)

var returnExportFields = []struct {
	Key   string
	Label string
}{
	{Key: "shop", Label: "店铺"},
	{Key: "productImage", Label: "商品图片"},
	{Key: "sku", Label: "规格"},
	{Key: "order", Label: "订单信息"},
	{Key: "aftersale", Label: "售后信息"},
	{Key: "logisticsNo", Label: "物流单号"},
	{Key: "returnLocation", Label: "退回地"},
	{Key: "fenFaRemark", Label: "分发备注"},
	{Key: "returnTime", Label: "物流退回时间"},
	{Key: "applyTime", Label: "申请时间"},
	{Key: "syncedAt", Label: "同步时间"},
}

func defaultReturnExportFields() []string {
	out := make([]string, 0, len(returnExportFields))
	for _, f := range returnExportFields {
		out = append(out, f.Key)
	}
	return out
}

func normalizeReturnExportFields(fields []string) []string {
	if len(fields) == 0 {
		return defaultReturnExportFields()
	}
	allow := map[string]struct{}{}
	for _, f := range returnExportFields {
		allow[f.Key] = struct{}{}
	}
	out := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, raw := range fields {
		k := strings.TrimSpace(raw)
		if _, ok := allow[k]; !ok {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	if len(out) == 0 {
		return defaultReturnExportFields()
	}
	return out
}

func fieldLabel(key string) string {
	for _, f := range returnExportFields {
		if f.Key == key {
			return f.Label
		}
	}
	return key
}

func LastTwoTracksText(tracks []dto.LogisticsTrack, fallback string) string {
	if len(tracks) == 0 {
		return strings.TrimSpace(fallback)
	}
	n := 2
	if len(tracks) < n {
		n = len(tracks)
	}
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		line := formatTrackLine(tracks[i])
		if line != "" {
			parts = append(parts, line)
		}
	}
	if len(parts) == 0 {
		return strings.TrimSpace(fallback)
	}
	return strings.Join(parts, "\n")
}

func formatTrackLine(t dto.LogisticsTrack) string {
	line := strings.TrimSpace(strings.Join([]string{t.Date, t.Title, t.Detail}, " "))
	if line != "" {
		return line
	}
	return strings.TrimSpace(t.Text)
}

func (s *ShopService) ExportReturns(q dto.ReturnExportRequest, bearerToken string) ([]byte, string, error) {
	list, _, err := s.ListReturns(dto.ReturnListQuery{
		ShopID:     q.ShopID,
		Keyword:    q.Keyword,
		ReturnFrom: q.ReturnFrom,
		ReturnTo:   q.ReturnTo,
		ApplyFrom:  q.ApplyFrom,
		ApplyTo:    q.ApplyTo,
		Page:       1,
		PageSize:   1500,
		Unpaged:    true,
	}, bearerToken)
	if err != nil {
		return nil, "", err
	}
	fields := normalizeReturnExportFields(q.Fields)
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	if sheet == "" {
		sheet = "Sheet1"
	}
	_ = f.SetSheetName(sheet, "退回管理")
	sheet = "退回管理"
	headers := make([]string, 0, len(fields))
	for _, key := range fields {
		headers = append(headers, fieldLabel(key))
	}
	if err := f.SetSheetRow(sheet, "A1", &headers); err != nil {
		return nil, "", err
	}
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	wrapStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	_ = f.SetRowHeight(sheet, 1, 22)
	_ = f.SetCellStyle(sheet, "A1", colName(len(fields))+"1", headerStyle)
	for i, key := range fields {
		width := 18.0
		switch key {
		case "productImage":
			width = 14
		case "sku":
			width = 28
		case "order", "aftersale", "returnLocation", "fenFaRemark":
			width = 32
		case "logisticsNo":
			width = 22
		}
		_ = f.SetColWidth(sheet, colName(i+1), colName(i+1), width)
	}
	imgHTTP := &http.Client{Timeout: 8 * time.Second}
	imgCache := map[string]fetchedExportImage{}
	embedImages := hasField(fields, "productImage")
	for i, row := range list {
		excelRow := i + 2
		values := make([]any, 0, len(fields))
		for _, key := range fields {
			values = append(values, exportFieldValue(row, key))
		}
		cell, _ := excelize.CoordinatesToCellName(1, excelRow)
		if err := f.SetSheetRow(sheet, cell, &values); err != nil {
			return nil, "", err
		}
		_ = f.SetRowHeight(sheet, excelRow, 72)
		_ = f.SetCellStyle(sheet, colName(1)+fmt.Sprint(excelRow), colName(len(fields))+fmt.Sprint(excelRow), wrapStyle)
		if embedImages {
			if pic := loadExportImage(imgHTTP, imgCache, row.ProductImage); pic.OK() {
				col := indexOfField(fields, "productImage") + 1
				picCell, _ := excelize.CoordinatesToCellName(col, excelRow)
				_ = addExportPicture(f, sheet, picCell, pic.Data, pic.Ext)
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, "", err
	}
	name := "退回管理-" + time.Now().Format("20060102-150405") + ".xlsx"
	return buf.Bytes(), name, nil
}

func exportFieldValue(row dto.ReturnPackageItem, key string) string {
	switch key {
	case "shop":
		return row.ShopName
	case "productImage":
		return ""
	case "sku":
		return strings.TrimSpace(row.SKU)
	case "order":
		return strings.TrimSpace(fmt.Sprintf("应付 ¥%s\n购买 %d 件\n订单 %s\n售后 %s",
			dash(row.PayAmount), orQty(row.BuyQty, row.Qty), dash(row.OrderNo), dash(row.PlatformAftersaleID)))
	case "aftersale":
		return strings.TrimSpace(fmt.Sprintf("%s\n售后退款 ¥%s\n申请 %d 件\n%s\n%s",
			dash(row.AftersaleType), dash(row.RefundAmount), row.Qty, prefixLine("申请原因 ", row.Reason), prefixLine("申请时间 ", row.ApplyTime)))
	case "logisticsNo":
		return strings.TrimSpace(row.LogisticsNo)
	case "returnLocation":
		return LastTwoTracksText(row.Tracks, row.ReturnLocation)
	case "fenFaRemark":
		return row.FenFaRemark
	case "returnTime":
		return row.ReturnTime
	case "applyTime":
		return row.ApplyTime
	case "syncedAt":
		return row.SyncedAt
	default:
		return ""
	}
}

func dash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	return s
}

func prefixLine(prefix, s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return prefix + s
}

func orQty(a, b int) int {
	if a > 0 {
		return a
	}
	return b
}

func hasField(fields []string, key string) bool {
	return indexOfField(fields, key) >= 0
}

func indexOfField(fields []string, key string) int {
	for i, f := range fields {
		if f == key {
			return i
		}
	}
	return -1
}

func colName(n int) string {
	name, _ := excelize.ColumnNumberToName(n)
	return name
}

type fetchedExportImage struct {
	Data []byte
	Ext  string
}

func (p fetchedExportImage) OK() bool { return len(p.Data) > 0 && p.Ext != "" }

func loadExportImage(httpClient *http.Client, cache map[string]fetchedExportImage, rawURL string) fetchedExportImage {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fetchedExportImage{}
	}
	if cached, ok := cache[rawURL]; ok {
		return cached
	}
	img, ext, err := fetchExportImage(httpClient, rawURL)
	out := fetchedExportImage{}
	if err == nil {
		out = fetchedExportImage{Data: img, Ext: ext}
	}
	cache[rawURL] = out
	return out
}

func addExportPicture(f *excelize.File, sheet, cell string, img []byte, ext string) error {
	img, ext, err := normalizeExcelImage(img, ext)
	if err != nil {
		return err
	}
	return f.AddPictureFromBytes(sheet, cell, &excelize.Picture{
		Extension: ext,
		File:      img,
		Format: &excelize.GraphicOptions{
			AutoFit:         true,
			LockAspectRatio: true,
		},
	})
}

var exportThumbSizeRe = regexp.MustCompile(`~48x\d+`)

func upgradeExportImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	return exportThumbSizeRe.ReplaceAllString(raw, "~240x240")
}

func fetchExportImage(httpClient *http.Client, rawURL string) ([]byte, string, error) {
	rawURL = upgradeExportImageURL(rawURL)
	if rawURL == "" || !(strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")) {
		return nil, "", fmt.Errorf("empty")
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://fxg.jinritemai.com/")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("http %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil || len(data) == 0 {
		return nil, "", fmt.Errorf("empty image")
	}
	hinted := strings.ToLower(path.Ext(strings.Split(rawURL, "?")[0]))
	if hinted == "" {
		ct := strings.ToLower(resp.Header.Get("Content-Type"))
		switch {
		case strings.Contains(ct, "png"):
			hinted = ".png"
		case strings.Contains(ct, "gif"):
			hinted = ".gif"
		case strings.Contains(ct, "webp"):
			hinted = ".webp"
		default:
			hinted = ".jpg"
		}
	}
	return normalizeExcelImage(data, hinted)
}

func detectImageKind(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return ".jpg"
	case bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}):
		return ".png"
	case bytes.HasPrefix(data, []byte("GIF8")):
		return ".gif"
	case bytes.HasPrefix(data, []byte("BM")):
		return ".bmp"
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return ".webp"
	default:
		return ""
	}
}

func normalizeExcelImage(data []byte, hintedExt string) ([]byte, string, error) {
	if len(data) == 0 {
		return nil, "", fmt.Errorf("empty image")
	}
	kind := detectImageKind(data)
	if kind == "" {
		kind = strings.ToLower(hintedExt)
	}
	switch kind {
	case ".jpg", ".jpeg":
		return data, ".jpg", nil
	case ".png", ".gif", ".bmp":
		return data, kind, nil
	case ".webp":
		img, err := webp.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, "", err
		}
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 88}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), ".jpg", nil
	default:
		return nil, "", fmt.Errorf("unsupported image %s", kind)
	}
}
