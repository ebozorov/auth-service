package user

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUserIDIsJSONString(t *testing.T) {
	b, err := json.Marshal(User{ID: 41784020480034629})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"id":"41784020480034629"`) {
		t.Fatalf("id is not a string: %s", b)
	}
}
