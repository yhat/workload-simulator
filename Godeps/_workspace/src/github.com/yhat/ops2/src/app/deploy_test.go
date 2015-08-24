package app

import "testing"

func TestValidModelName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{"foobar", true},
		{"foobar ", false},
		{"", false},
		{`
foobar
`, false},
		{"hello_world0813", true},
		{"hello_world081 3", false},
	}

	for _, test := range tests {
		isValid := isValidModelName(test.name)
		if isValid && !test.ok {
			t.Errorf("expected '%s' to be an invalid model name", test.name)
		} else if !isValid && test.ok {
			t.Errorf("expected '%s' to be an valid model name", test.name)
		}
	}
}
