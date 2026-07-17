package main

import "testing"

func TestVersion(t *testing.T) {
	if version != "1.0.0" {
		t.Fatalf("version = %q", version)
	}
}
