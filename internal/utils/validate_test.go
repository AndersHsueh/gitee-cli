package utils

import "testing"

func TestValidateName(t *testing.T) {
	cases := []struct {
		noun  string
		name  string
		valid bool
	}{
		{"alias", "", false},
		{"alias", ".hidden", false},
		{"alias", "-flag", false},
		{"alias", "with space", false},
		{"alias", "UPPER", false},
		{"alias", "with.dot", false},
		{"alias", "valid_name", true},
		{"alias", "valid-name", true},
		{"alias", "a-1_b", true},
		{"extension", "", false},
		{"extension", "valid", true},
	}
	for _, tc := range cases {
		err := ValidateName(tc.noun, tc.name)
		got := err == nil
		if got != tc.valid {
			t.Errorf("ValidateName(%q, %q) err=%v, want valid=%v", tc.noun, tc.name, err, tc.valid)
		}
	}
}

func TestValidateNameMaxLen(t *testing.T) {
	name41 := make([]byte, MaxNameLen+1)
	for i := range name41 {
		name41[i] = 'a'
	}
	if err := ValidateName("alias", string(name41)); err == nil {
		t.Error("name longer than MaxNameLen should error")
	}
	name40 := make([]byte, MaxNameLen)
	for i := range name40 {
		name40[i] = 'a'
	}
	if err := ValidateName("alias", string(name40)); err != nil {
		t.Errorf("name exactly MaxNameLen should pass, got: %v", err)
	}
}
