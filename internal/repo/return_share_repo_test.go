package repo

import "testing"

func TestSplitReturnLocationFilters(t *testing.T) {
	got := SplitReturnLocationFilters(" 宿迁, 昆山；花桥\n花桥 ")
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	if got[0] != "宿迁" || got[1] != "昆山" || got[2] != "花桥" {
		t.Fatalf("got %v", got)
	}
	if SplitReturnLocationFilters("   ") != nil {
		t.Fatal("empty should be nil")
	}
}
