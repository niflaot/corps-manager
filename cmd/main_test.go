package main

import "testing"

func TestVersion(t *testing.T) {
	if version != "1.1.1" {
		t.Fatalf("version = %q", version)
	}
}
