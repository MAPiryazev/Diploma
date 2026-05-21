package security

import "testing"

func TestMaskID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{name: "short id masked fully", id: "12345678", want: "****"},
		{name: "trimmed short id masked fully", id: " 1234 ", want: "****"},
		{name: "long id keeps edges", id: "1234567890abcdef", want: "1234...cdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskID(tt.id); got != tt.want {
				t.Fatalf("MaskID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestMaskAmount(t *testing.T) {
	for _, amount := range []string{"", "100.00", " 2500 "} {
		if got := MaskAmount(amount); got != "***" {
			t.Fatalf("MaskAmount(%q) = %q, want %q", amount, got, "***")
		}
	}
}
