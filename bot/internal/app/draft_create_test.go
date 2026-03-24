package app

import (
	"fmt"
	"testing"

	"bot/internal/erp"
)

func TestCreateDraftWithFreshEPC_RetriesDuplicateAndSucceeds(t *testing.T) {
	epcs := []string{"EPC-1", "EPC-2"}
	nextIdx := 0

	epc, draft, err := createDraftWithFreshEPC(
		func() string {
			v := epcs[nextIdx]
			nextIdx++
			return v
		},
		func(epc string) (erp.StockEntryDraft, error) {
			if epc == "EPC-1" {
				return erp.StockEntryDraft{}, fmt.Errorf("erp stock entry http 417: barcode already exists")
			}
			return erp.StockEntryDraft{Name: "MAT-STE-1", Barcode: epc}, nil
		},
	)
	if err != nil {
		t.Fatalf("createDraftWithFreshEPC error: %v", err)
	}
	if epc != "EPC-2" {
		t.Fatalf("epc mismatch: %q", epc)
	}
	if draft.Barcode != "EPC-2" {
		t.Fatalf("draft barcode mismatch: %q", draft.Barcode)
	}
}

func TestCreateDraftWithFreshEPC_ReturnsNonDuplicateImmediately(t *testing.T) {
	_, _, err := createDraftWithFreshEPC(
		func() string { return "EPC-1" },
		func(epc string) (erp.StockEntryDraft, error) {
			return erp.StockEntryDraft{}, fmt.Errorf("erp stock entry http 500: unexpected failure")
		},
	)
	if err == nil || err.Error() != "erp stock entry http 500: unexpected failure" {
		t.Fatalf("unexpected error: %v", err)
	}
}
