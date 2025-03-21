package auth

import (
	"testing"
)

func Test_stringToAES192Key(t *testing.T) {

	tests := []struct {
		name         string
		str          string
		wantBytesLen int
	}{
		{
			name:         "test",
			str:          "test",
			wantBytesLen: 192 / 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HashToAES192Key([]byte(tt.str)); len(got) != tt.wantBytesLen {
				t.Errorf("stringToAES192Key() len = %v, want %v", len(got), tt.wantBytesLen)
			}
		})
	}
}
