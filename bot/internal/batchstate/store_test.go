package batchstate

import (
	bridgestate "bridge/state"
	"path/filepath"
	"testing"
)

func TestSetWritesSnapshot(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "bridge_state.json")

	s := New(p)
	if err := s.Set(true, 123, "ITM-001", "GRENKI YASHIL", "Stores - A"); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	got, err := bridgestate.New(p).Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if !got.Batch.Active {
		t.Fatalf("active mismatch: %v", got.Batch.Active)
	}
	if got.Batch.ChatID != 123 {
		t.Fatalf("chat_id mismatch: %v", got.Batch.ChatID)
	}
	if got.Batch.ItemCode != "ITM-001" {
		t.Fatalf("item_code mismatch: %q", got.Batch.ItemCode)
	}
	if got.Batch.ItemName != "GRENKI YASHIL" {
		t.Fatalf("item_name mismatch: %q", got.Batch.ItemName)
	}
	if got.Batch.Warehouse != "Stores - A" {
		t.Fatalf("warehouse mismatch: %q", got.Batch.Warehouse)
	}
	if got.Batch.UpdatedAt == "" {
		t.Fatalf("updated_at missing")
	}
}

func TestSetInactiveClearsItemFields(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "bridge_state.json")

	s := New(p)
	if err := s.Set(true, 123, "ITM-001", "ITEM", "Stores - A"); err != nil {
		t.Fatalf("Set active error: %v", err)
	}
	if err := s.SetPrintRequest("3034257BF7194E406994036B", 2.5, "kg", "ITM-001", "ITEM"); err != nil {
		t.Fatalf("SetPrintRequest error: %v", err)
	}
	if err := s.Set(false, 123, "", "", ""); err != nil {
		t.Fatalf("Set inactive error: %v", err)
	}

	got, err := bridgestate.New(p).Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if got.Batch.Active {
		t.Fatalf("active should be false")
	}
	if got.Batch.ItemCode != "" || got.Batch.ItemName != "" || got.Batch.Warehouse != "" {
		t.Fatalf("item fields not cleared: %+v", got.Batch)
	}
	if got.PrintRequest.EPC != "" || got.PrintRequest.Status != "" || got.PrintRequest.Qty != nil {
		t.Fatalf("print request should be cleared on inactive batch: %+v", got.PrintRequest)
	}
}

func TestSetPrintRequestWritesSnapshot(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "bridge_state.json")

	s := New(p)
	if err := s.SetPrintRequest("3034257bf7194e406994036b", 2.5, "kg", "ITM-001", "GREEN TEA"); err != nil {
		t.Fatalf("SetPrintRequest error: %v", err)
	}

	got, err := bridgestate.New(p).Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if got.PrintRequest.EPC != "3034257BF7194E406994036B" {
		t.Fatalf("print request epc mismatch: %q", got.PrintRequest.EPC)
	}
	if got.PrintRequest.Qty == nil || *got.PrintRequest.Qty != 2.5 {
		t.Fatalf("print request qty mismatch: %+v", got.PrintRequest.Qty)
	}
	if got.PrintRequest.Unit != "kg" {
		t.Fatalf("print request unit mismatch: %q", got.PrintRequest.Unit)
	}
	if got.PrintRequest.ItemCode != "ITM-001" {
		t.Fatalf("print request item code mismatch: %q", got.PrintRequest.ItemCode)
	}
	if got.PrintRequest.ItemName != "GREEN TEA" {
		t.Fatalf("print request item name mismatch: %q", got.PrintRequest.ItemName)
	}
	if got.PrintRequest.Status != "pending" {
		t.Fatalf("print request status mismatch: %q", got.PrintRequest.Status)
	}
	if got.PrintRequest.RequestedAt == "" || got.PrintRequest.UpdatedAt == "" {
		t.Fatalf("print request timestamps missing: %+v", got.PrintRequest)
	}
}

func TestClearPrintRequestClearsFields(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "bridge_state.json")

	s := New(p)
	if err := s.SetPrintRequest("3034257BF7194E406994036B", 2.5, "kg", "ITM-001", "GREEN TEA"); err != nil {
		t.Fatalf("SetPrintRequest error: %v", err)
	}
	if err := s.ClearPrintRequest(); err != nil {
		t.Fatalf("ClearPrintRequest error: %v", err)
	}

	got, err := bridgestate.New(p).Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if got.PrintRequest.EPC != "" || got.PrintRequest.Status != "" || got.PrintRequest.ItemCode != "" {
		t.Fatalf("print request not cleared: %+v", got.PrintRequest)
	}
	if got.PrintRequest.Qty != nil {
		t.Fatalf("print request qty should be nil: %+v", got.PrintRequest.Qty)
	}
}
