package security

import (
	"strings"
	"testing"
)

func TestValidateCommand(t *testing.T) {
	for _, tt := range []struct {
		name    string
		wantErr bool
	}{
		{"open_spotify", false},
		{"system_status", false},
		{"", true},
		{strings.Repeat("x", 65), true},
	} {
		err := ValidateCommand(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateCommand(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestValidateQuery(t *testing.T) {
	for _, tt := range []struct {
		query   string
		wantErr bool
	}{
		{"go closures", false},
		{"authentication middleware", false},
		{"", true},
		{"   ", true},
		{strings.Repeat("a", MaxQueryLength+1), true},
	} {
		err := ValidateQuery(tt.query)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateQuery(%q) error = %v, wantErr %v", tt.query, err, tt.wantErr)
		}
	}
}
