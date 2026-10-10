package service

import "testing"

func TestPickFenFaRemarkBySKU(t *testing.T) {
	pid := "6926674711466049482"
	remarks := map[string]string{
		fenFaRemarkLookupKey(pid, "[公路压入]BB72 中轴"):                     "390",
		fenFaRemarkLookupKey(pid, "CL800 140/160mm各一张（配内锁盖）"):          "390",
		fenFaRemarkLookupKey(pid, "【电变版】R7170 散热大套（牙盘50-34T/170）不含中轴"): "刘沛代发 暂时不发",
		// 多销售单备注不一致时，订单中心不再回填裸平台单号
	}
	if got := pickFenFaRemark(remarks, pid, "[公路压入]BB72 中轴"); got != "390" {
		t.Fatalf("bb72 got %q", got)
	}
	if got := pickFenFaRemark(remarks, pid, "CL800 140/160mm各一张（配内锁盖）"); got != "390" {
		t.Fatalf("cl800 got %q", got)
	}
	if got := pickFenFaRemark(remarks, pid, ""); got != "" {
		t.Fatalf("no sku should not invent remark, got %q", got)
	}
	remarks[pid] = "390"
	if got := pickFenFaRemark(remarks, pid, "未知规格"); got != "390" {
		t.Fatalf("fallback bare platform got %q", got)
	}
}
