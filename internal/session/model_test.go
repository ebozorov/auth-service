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
