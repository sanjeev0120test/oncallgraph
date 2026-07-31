package main

import (
	"errors"
	"testing"
)

func TestExitCodeFor(t *testing.T) {
	if exitCodeFor(nil) != 0 {
		t.Fatal("nil => 0")
	}
	if exitCodeFor(fail(1, "not found")) != 1 {
		t.Fatal("fail(1) => 1")
	}
	if exitCodeFor(fail(2, "bad config")) != 2 {
		t.Fatal("fail(2) => 2")
	}
	if exitCodeFor(errors.New("other")) != 2 {
		t.Fatal("unknown => 2")
	}
}
