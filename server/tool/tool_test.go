package tool

import (
	"crypto/rand"
	"encoding/base64"
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

func BenchmarkHashToAES192Key(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HashToAES192Key2([]byte("11111111"))
	}
}

func BenchmarkAES192KeyKDF(b *testing.B) {
	c := "11111111"
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = AES192KeyKDF(c, salt)
	}
}

func TestAES192KeyKDF(t *testing.T) {
	const pwd = "mysecretpassword"
	const salt = "test"
	kdf := AES192KeyKDF(pwd, []byte(salt))
	kdfB64 := base64.StdEncoding.EncodeToString(kdf)
	expected := "9Dt2Ws9OB1uDRkxK4IHBHpqm9rMQ0d+z"
	if kdfB64 != expected {
		t.Errorf("AES192KeyKDF() = %v, want %v", kdfB64, expected)
	}
}
