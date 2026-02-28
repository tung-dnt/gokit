package user

import (
	"encoding/hex"
	"testing"
)

func TestGenerateID(t *testing.T) {
	t.Parallel()

	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID() error = %v", err)
	}
	if len(id) != 16 {
		t.Errorf("expected 16-char string, got %q (len %d)", id, len(id))
	}
	if _, decErr := hex.DecodeString(id); decErr != nil {
		t.Errorf("expected valid hex string, got %q: %v", id, decErr)
	}
}

func TestGenerateID_Unique(t *testing.T) {
	t.Parallel()

	id1, _ := generateID()
	id2, _ := generateID()
	if id1 == id2 {
		t.Errorf("expected unique IDs, got %q twice", id1)
	}
}
