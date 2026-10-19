package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResponseIDIsJSONString(t *testing.T) {
	b, err := json.Marshal(Response{ID: 41784020480034629})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"id":"41784020480034629"`) {
		t.Fatalf("id is not a string: %s", b)
	}
}

func TestNewResponsesMarksCurrent(t *testing.T) {
	got := NewResponses([]Session{{ID: 1}, {ID: 2}, {ID: 3}}, 2)
	if len(got) != 3 {
		t.Fatalf("got %d", len(got))
	}
	for _, r := range got {
		if r.Current != (r.ID == 2) {
			t.Errorf("session %d: current %v", r.ID, r.Current)
		}
	}
}

// No sessions is an empty JSON list, not null.
func TestNewResponsesEmptyIsList(t *testing.T) {
	b, err := json.Marshal(NewResponses(nil, 1))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "[]" {
		t.Fatalf("got %s", b)
	}
}

func TestEndedJSON(t *testing.T) {
	b, err := json.Marshal(EndedBy(ReasonTerminated, 41784020480034629, 7))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"reason":"terminated"`, `"by":41784020480034629`, `"by_session":7`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("%s does not contain %s", b, want)
		}
	}

	// Ended by the system: no user, no session.
	b, err = json.Marshal(Ended{Reason: ReasonExpired})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"by":null,"by_session":null`) {
		t.Fatalf("got %s", b)
	}
}
