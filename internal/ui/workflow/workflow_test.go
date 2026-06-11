package workflow

import (
	"testing"

	domainsession "aws-terminal/internal/domain/session"
	"aws-terminal/internal/ui/pageapi"
)

func TestActiveRegionPrefersSelectedRegion(t *testing.T) {
	state := pageapi.State{
		SelectedRegion: "eu-west-1",
		ActiveSession:  &domainsession.Session{Region: "us-east-1"},
	}

	if got := ActiveRegion(state); got != "eu-west-1" {
		t.Fatalf("expected selected region, got %q", got)
	}
}

func TestSessionKeyUsesProfileAndActiveRegion(t *testing.T) {
	state := pageapi.State{
		SelectedRegion: "eu-west-1",
		ActiveSession:  &domainsession.Session{Profile: "prod", Region: "us-east-1"},
	}

	if got := SessionKey(state); got != "prod|eu-west-1" {
		t.Fatalf("expected session key, got %q", got)
	}
}

func TestFirstStatusUsesFirstActiveCandidate(t *testing.T) {
	status := FirstStatus(
		Error(""),
		Activity(false, "inactive"),
		Activity(true, "loading"),
		Error("later error"),
	)

	if status.Message != "loading" || status.Error != "" {
		t.Fatalf("expected loading message, got %#v", status)
	}
}

func TestFirstStatusCandidateErrorWinsOverCandidateMessage(t *testing.T) {
	status := FirstStatus(Candidate{Active: true, Message: "loading", Error: "failed"})
	if status.Error != "failed" || status.Message != "" {
		t.Fatalf("expected error status, got %#v", status)
	}
}
