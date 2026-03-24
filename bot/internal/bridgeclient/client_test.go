package bridgeclient

import (
	bridgestate "bridge/state"
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitStablePositiveReading_Stable(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "bridge_state.json")
	s := bridgestate.New(p)
	w := 1.234
	st := true
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := s.Update(func(snapshot *bridgestate.Snapshot) {
		snapshot.Scale.Weight = &w
		snapshot.Scale.Stable = &st
		snapshot.Scale.Unit = "kg"
		snapshot.Scale.UpdatedAt = now
	}); err != nil {
		t.Fatal(err)
	}

	c := New(p)
	r, err := c.WaitStablePositiveReading(context.Background(), 2*time.Second, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitStablePositiveReading error: %v", err)
	}
	if r.Qty != 1.234 {
		t.Fatalf("qty mismatch: %v", r.Qty)
	}
	if r.Unit != "kg" {
		t.Fatalf("unit mismatch: %q", r.Unit)
	}
}

func TestWaitForNextCycle_ReturnsOnMeaningfulChange(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "bridge_state.json")
	s := bridgestate.New(p)

	w := 10.000
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := s.Update(func(snapshot *bridgestate.Snapshot) {
		snapshot.Scale.Weight = &w
		snapshot.Scale.UpdatedAt = now
	}); err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(120 * time.Millisecond)
		w2 := 10.020
		_ = s.Update(func(snapshot *bridgestate.Snapshot) {
			snapshot.Scale.Weight = &w2
			snapshot.Scale.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		})
	}()

	c := New(p)
	if err := c.WaitForNextCycle(context.Background(), 2*time.Second, 40*time.Millisecond, 10.000); err != nil {
		t.Fatalf("WaitForNextCycle error: %v", err)
	}
}

func TestWaitForNextCycle_IgnoresTinyJitter(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "bridge_state.json")
	s := bridgestate.New(p)

	w := 10.000
	if err := s.Update(func(snapshot *bridgestate.Snapshot) {
		snapshot.Scale.Weight = &w
		snapshot.Scale.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}); err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(90 * time.Millisecond)
		w2 := 10.003
		_ = s.Update(func(snapshot *bridgestate.Snapshot) {
			snapshot.Scale.Weight = &w2
			snapshot.Scale.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		})
		time.Sleep(90 * time.Millisecond)
		w3 := 10.000
		_ = s.Update(func(snapshot *bridgestate.Snapshot) {
			snapshot.Scale.Weight = &w3
			snapshot.Scale.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		})
	}()

	c := New(p)
	err := c.WaitForNextCycle(context.Background(), 350*time.Millisecond, 40*time.Millisecond, 10.000)
	if err == nil {
		t.Fatal("expected timeout, got nil")
	}
}

func TestWaitPrintRequestResult(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "bridge_state.json")
	s := bridgestate.New(p)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := s.Update(func(snapshot *bridgestate.Snapshot) {
		snapshot.PrintRequest.EPC = "3034257BF7194E406994036B"
		snapshot.PrintRequest.Status = "done"
		snapshot.PrintRequest.UpdatedAt = now
	}); err != nil {
		t.Fatal(err)
	}

	c := New(p)
	got, err := c.WaitPrintRequestResult(context.Background(), 500*time.Millisecond, 50*time.Millisecond, "3034257BF7194E406994036B")
	if err != nil {
		t.Fatalf("WaitPrintRequestResult error: %v", err)
	}
	if got.EPC != "3034257BF7194E406994036B" {
		t.Fatalf("epc mismatch: %q", got.EPC)
	}
	if got.Status != "done" {
		t.Fatalf("status mismatch: %q", got.Status)
	}
}
