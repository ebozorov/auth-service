package auth

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClaimsSIDIsString(t *testing.T) {
	b, err := json.Marshal(Claims{SessionID: 41784020480034629})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"sid":"41784020480034629"`) {
		t.Fatalf("sid is not a string: %s", b)
	}
}

// Tokens issued before sid became a string carry it as a number; they stay valid until they expire.
func TestClaimsSIDAcceptsNumber(t *testing.T) {
	var c Claims
	if err := json.Unmarshal([]byte(`{"sid":41784020480034629}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.SessionID != 41784020480034629 {
		t.Fatalf("got %d", c.SessionID)
	}
}
