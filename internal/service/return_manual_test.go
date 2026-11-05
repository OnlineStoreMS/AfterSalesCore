package service

import (
	"testing"

	"aftersalescore/internal/model"
)

func TestIsManualReturn(t *testing.T) {
	if isManualReturn(nil) {
		t.Fatal("nil should not be manual")
	}
	if isManualReturn(&model.ReturnPackage{RawJSON: `{"foo":1}`}) {
		t.Fatal("collected payload should not be manual")
	}
	if !isManualReturn(&model.ReturnPackage{RawJSON: `{"source":"manual"}`}) {
		t.Fatal("manual source should be recognized")
	}
}
