package user

import "testing"

func TestNormalizeEmail(t *testing.T) {
	for in, want := range map[string]string{
		"bob@example.com":       "bob@example.com",
		"  Bob@Example.COM \n":  "bob@example.com",
		"":                      "",
		"\tALICE@EXAMPLE.COM\t": "alice@example.com",
	} {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

func TestIsValidRole(t *testing.T) {
	for role, want := range map[string]bool{
		RoleAdmin: true,
		RoleUser:  true,
		"":        false,
		"Admin":   false,
		"root":    false,
	} {
		if got := IsValidRole(role); got != want {
			t.Errorf("%q: got %v, want %v", role, got, want)
		}
	}
}
