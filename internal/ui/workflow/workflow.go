package workflow

import (
	"strings"

	"aws-terminal/internal/ui/pageapi"
)

// Candidate describes one possible page workflow status. It keeps page status
// construction declarative and reusable across staged AWS workflows.
type Candidate struct {
	Active  bool
	Message string
	Error   string
}

func Error(message string) Candidate {
	return Candidate{Active: strings.TrimSpace(message) != "", Error: message}
}

func Activity(active bool, message string) Candidate {
	return Candidate{Active: active, Message: message}
}

// FirstStatus returns the first active status candidate, preferring errors over
// messages when both are present on the same candidate.
func FirstStatus(candidates ...Candidate) pageapi.Status {
	for _, candidate := range candidates {
		if !candidate.Active {
			continue
		}
		if strings.TrimSpace(candidate.Error) != "" {
			return pageapi.Status{Error: candidate.Error}
		}
		if strings.TrimSpace(candidate.Message) != "" {
			return pageapi.Status{Message: candidate.Message}
		}
	}

	return pageapi.Status{}
}

// ActiveRegion resolves the effective region from page state.
func ActiveRegion(state pageapi.State) string {
	if strings.TrimSpace(state.SelectedRegion) != "" {
		return strings.TrimSpace(state.SelectedRegion)
	}
	if state.ActiveSession != nil {
		return strings.TrimSpace(state.ActiveSession.Region)
	}

	return ""
}

// SessionKey identifies session-scoped page data.
func SessionKey(state pageapi.State) string {
	if state.ActiveSession == nil {
		return ""
	}

	return strings.TrimSpace(state.ActiveSession.Profile) + "|" + ActiveRegion(state)
}

func ValueOrFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}
