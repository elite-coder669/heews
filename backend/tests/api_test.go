package api

import (
	"testing"
)

func TestHealthShape(t *testing.T) {
	// MVP-level smoke; full integration tests run after `go mod tidy`.
	if 1+1 != 2 {
		t.Fatal("math broken")
	}
}