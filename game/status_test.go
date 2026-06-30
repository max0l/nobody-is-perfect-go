package game

import "testing"

func TestGameStatusString(t *testing.T) {
	tests := map[GameStatus]string{
		GameStatusCreated:  "created",
		GameStatusStarted:  "started",
		GameStatusFinished: "finished",
		GameStatus(99):     "unknown",
	}

	for status, want := range tests {
		if got := status.String(); got != want {
			t.Fatalf("expected %q for %d, got %q", want, status, got)
		}
	}
}

func TestGameStatusIsValid(t *testing.T) {
	if !GameStatusCreated.IsValid() || !GameStatusStarted.IsValid() || !GameStatusFinished.IsValid() {
		t.Fatal("expected defined statuses to be valid")
	}
	if GameStatus(99).IsValid() {
		t.Fatal("expected unknown status to be invalid")
	}
}

func TestRoundStatusString(t *testing.T) {
	tests := map[RoundStatus]string{
		RoundStatusAnswering: "answering",
		RoundStatusVerifying: "verifying",
		RoundStatusVoting:    "voting",
		RoundStatusRevealed:  "revealed",
		RoundStatus(99):      "unknown",
	}

	for status, want := range tests {
		if got := status.String(); got != want {
			t.Fatalf("expected %q for %d, got %q", want, status, got)
		}
	}
}

func TestRoundStatusIsValid(t *testing.T) {
	if !RoundStatusAnswering.IsValid() || !RoundStatusVerifying.IsValid() || !RoundStatusVoting.IsValid() || !RoundStatusRevealed.IsValid() {
		t.Fatal("expected defined round statuses to be valid")
	}
	if RoundStatus(99).IsValid() {
		t.Fatal("expected unknown round status to be invalid")
	}
}
