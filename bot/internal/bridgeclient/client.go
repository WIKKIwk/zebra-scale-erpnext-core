package bridgeclient

import (
	bridgestate "bridge/state"
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// nextCycleDeltaEpsilon: WaitForNextCycle yangi siklni ochish uchun
// oxirgi qayd qilingan qty dan minimal ma'noli o'zgarish (kg).
// Juda kichik jitterlar yangi sikl ochmasligi uchun ishlatiladi.
const nextCycleDeltaEpsilon = 0.005

type Client struct {
	store *bridgestate.Store
}

type StableReading struct {
	Qty       float64
	Unit      string
	UpdatedAt time.Time
}

type PrintRequestResult struct {
	EPC       string
	Status    string
	Error     string
	UpdatedAt time.Time
}

func New(path string) *Client {
	return &Client{store: bridgestate.New(path)}
}

func (c *Client) WaitStablePositive(ctx context.Context, timeout, pollInterval time.Duration) (float64, string, error) {
	r, err := c.WaitStablePositiveReading(ctx, timeout, pollInterval)
	if err != nil {
		return 0, "", err
	}
	return r.Qty, r.Unit, nil
}

func (c *Client) WaitStablePositiveReading(ctx context.Context, timeout, pollInterval time.Duration) (StableReading, error) {
	if c == nil || c.store == nil || strings.TrimSpace(c.store.Path()) == "" {
		return StableReading{}, fmt.Errorf("bridge state path bo'sh")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if pollInterval <= 0 {
		pollInterval = 220 * time.Millisecond
	}

	deadline := time.Now().Add(timeout)
	var lastWeight float64
	var haveLast bool
	stableCount := 0

	for {
		if time.Now().After(deadline) {
			return StableReading{}, fmt.Errorf("scale qty timeout (%s)", timeout)
		}
		select {
		case <-ctx.Done():
			return StableReading{}, ctx.Err()
		default:
		}

		snap, err := c.store.Read()
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}
		s := snap.Scale
		if strings.TrimSpace(s.Error) != "" {
			haveLast = false
			stableCount = 0
			time.Sleep(pollInterval)
			continue
		}
		if s.Weight == nil || *s.Weight <= 0 {
			haveLast = false
			stableCount = 0
			time.Sleep(pollInterval)
			continue
		}
		updatedAt, ok := parseSnapshotTime(s.UpdatedAt)
		if !ok || !isFreshTime(updatedAt, 4*time.Second) {
			time.Sleep(pollInterval)
			continue
		}

		w := *s.Weight
		if s.Stable != nil && *s.Stable {
			return StableReading{Qty: w, Unit: normalizeUnit(s.Unit), UpdatedAt: updatedAt}, nil
		}

		if haveLast && almostEqual(lastWeight, w, 0.001) {
			stableCount++
		} else {
			stableCount = 1
		}
		haveLast = true
		lastWeight = w

		if stableCount >= 4 {
			return StableReading{Qty: w, Unit: normalizeUnit(s.Unit), UpdatedAt: updatedAt}, nil
		}
		time.Sleep(pollInterval)
	}
}

func (c *Client) WaitPrintRequestResult(ctx context.Context, timeout, pollInterval time.Duration, epc string) (PrintRequestResult, error) {
	if c == nil || c.store == nil || strings.TrimSpace(c.store.Path()) == "" {
		return PrintRequestResult{}, fmt.Errorf("bridge state path bo'sh")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if pollInterval <= 0 {
		pollInterval = 120 * time.Millisecond
	}
	epc = strings.ToUpper(strings.TrimSpace(epc))
	if epc == "" {
		return PrintRequestResult{}, fmt.Errorf("print request epc bo'sh")
	}

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return PrintRequestResult{}, fmt.Errorf("print request timeout (%s)", timeout)
		}
		select {
		case <-ctx.Done():
			return PrintRequestResult{}, ctx.Err()
		default:
		}

		snap, err := c.store.Read()
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		req := snap.PrintRequest
		gotEPC := strings.ToUpper(strings.TrimSpace(req.EPC))
		if gotEPC != epc {
			time.Sleep(pollInterval)
			continue
		}

		status := strings.ToLower(strings.TrimSpace(req.Status))
		if status != "done" && status != "error" {
			time.Sleep(pollInterval)
			continue
		}

		at, _ := parseSnapshotTime(req.UpdatedAt)
		return PrintRequestResult{
			EPC:       gotEPC,
			Status:    status,
			Error:     strings.TrimSpace(req.Error),
			UpdatedAt: at,
		}, nil
	}
}

// WaitForNextCycle returns when scale goes to reset (<=0) OR weight
// last processed qty dan ma'noli o'zgaradi (epsilon dan katta).
func (c *Client) WaitForNextCycle(ctx context.Context, timeout, pollInterval time.Duration, lastQty float64) error {
	if c == nil || c.store == nil || strings.TrimSpace(c.store.Path()) == "" {
		return fmt.Errorf("bridge state path bo'sh")
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	if pollInterval <= 0 {
		pollInterval = 220 * time.Millisecond
	}

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("scale next-cycle timeout (%s)", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		snap, err := c.store.Read()
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}
		s := snap.Scale
		if !isFreshSnapshot(s.UpdatedAt, 4*time.Second) {
			time.Sleep(pollInterval)
			continue
		}
		if s.Weight == nil || *s.Weight <= 0 {
			return nil
		}
		// Oxirgi qty dan ma'noli og'ish bo'lsa yangi sikl boshlangan deb olamiz.
		if lastQty > 0 && math.Abs(*s.Weight-lastQty) > nextCycleDeltaEpsilon {
			return nil
		}

		time.Sleep(pollInterval)
	}
}

func isFreshSnapshot(updated string, maxAge time.Duration) bool {
	ts, ok := parseSnapshotTime(updated)
	if !ok {
		return false
	}
	return isFreshTime(ts, maxAge)
}

func parseSnapshotTime(updated string) (time.Time, bool) {
	updated = strings.TrimSpace(updated)
	if updated == "" {
		return time.Time{}, false
	}
	ts, err := time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

func isFreshTime(ts time.Time, maxAge time.Duration) bool {
	age := time.Since(ts)
	if age < 0 {
		age = 0
	}
	return age <= maxAge
}

func normalizeUnit(v string) string {
	u := strings.TrimSpace(v)
	if u == "" {
		return "kg"
	}
	return u
}

func almostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}
