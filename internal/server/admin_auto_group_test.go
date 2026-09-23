package server

import "testing"

func TestValidateAutoGroupUserMax(t *testing.T) {
	cases := []struct {
		name  string
		value any
		valid bool
	}{
		{"minimum", 5, true},
		{"maximum", "10000", true},
		{"below minimum", 4, false},
		{"above maximum", 10001, false},
		{"fraction", 1.5, false},
		{"not a number", "abc", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAutoGroupUserMax(tc.value)
			if (err == nil) != tc.valid {
				t.Fatalf("validateAutoGroupUserMax(%v) error = %v, valid = %v", tc.value, err, tc.valid)
			}
		})
	}
}
