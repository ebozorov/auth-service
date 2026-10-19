package internal

import (
	"encoding/json"
	"testing"
)

// big is above 2^53, where a JSON number loses precision in JavaScript.
const big = 41784020480034629

func TestIDMarshalIsString(t *testing.T) {
	b, err := json.Marshal(struct {
		ID ID `json:"id"`
	}{big})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), `{"id":"41784020480034629"}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestIDUnmarshalAcceptsStringAndNumber(t *testing.T) {
	for _, in := range []string{`"41784020480034629"`, `41784020480034629`} {
		var id ID
		if err := json.Unmarshal([]byte(in), &id); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if id != big {
			t.Fatalf("%s: got %d", in, id)
		}
	}
}

func TestIDUnmarshalNullKeepsValue(t *testing.T) {
	id := ID(7)
	if err := json.Unmarshal([]byte(`null`), &id); err != nil || id != 7 {
		t.Fatalf("got %d, %v", id, err)
	}
}

func TestIDUnmarshalRejectsNonInteger(t *testing.T) {
	for _, in := range []string{`"abc"`, `"1.5"`, `1.5`, `""`, `true`, `"99999999999999999999"`} {
		var id ID
		if err := json.Unmarshal([]byte(in), &id); err == nil {
			t.Fatalf("%s: expected error, got %d", in, id)
		}
	}
}

func TestNewIDIsUniqueAndOrdered(t *testing.T) {
	if err := InitID(); err != nil {
		t.Skipf("no private IP for the machine id: %v", err)
	}
	prev := NewID()
	for range 1000 {
		id := NewID()
		if id <= prev {
			t.Fatalf("id %d is not after %d", id, prev)
		}
		prev = id
	}
}
