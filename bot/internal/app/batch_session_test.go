package app

import "testing"

func TestOtherActiveBatchOwner(t *testing.T) {
	a := &App{
		batchByChat: map[int64]batchSession{
			1001: {id: 1},
		},
	}

	if owner, ok := a.otherActiveBatchOwner(1001); ok || owner != 0 {
		t.Fatalf("same chat should not be treated as other active owner: owner=%d ok=%v", owner, ok)
	}

	owner, ok := a.otherActiveBatchOwner(2002)
	if !ok {
		t.Fatal("expected another active owner")
	}
	if owner != 1001 {
		t.Fatalf("owner mismatch: got=%d want=%d", owner, 1001)
	}
}
