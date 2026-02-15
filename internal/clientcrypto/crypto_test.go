package clientcrypto

import (
	"bytes"
	"testing"

	"github.com/makimaki04/go-data-keeper.git/pkg/contract"
)

func TestDeriveKey(t *testing.T) {
	t.Parallel()

	validSalt := bytes.Repeat([]byte{0xAB}, 16)
	validParams := contract.Params{
		Algorithm:   "argon2id",
		KeyLen:      32,
		SaltLen:     16,
		Time:        1,
		Memory:      8, // small for tests
		Parallelism: 1,
	}

	tests := []struct {
		name    string
		salt    []byte
		params  contract.Params
		wantErr bool
	}{
		{
			name:    "ok",
			salt:    validSalt,
			params:  validParams,
			wantErr: false,
		},
		{
			name:    "bad_salt_len",
			salt:    []byte{1, 2, 3},
			params:  validParams,
			wantErr: true,
		},
		{
			name: "bad_parallelism",
			salt: validSalt,
			params: contract.Params{
				Algorithm:   "argon2id",
				KeyLen:      32,
				SaltLen:     16,
				Time:        1,
				Memory:      8,
				Parallelism: 256,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key, err := DeriveKey("passw0rd", tt.salt, tt.params)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DeriveKey() err=%v wantErr=%v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(key) != int(tt.params.KeyLen) {
				t.Fatalf("unexpected key len: got=%d want=%d", len(key), tt.params.KeyLen)
			}
		})
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{0x11}, 32)
	plaintext := []byte("hello")
	aad := []byte("aad")

	ciphertext, nonce, err := Encrypt(key, plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt() err=%v", err)
	}
	if len(ciphertext) == 0 || len(nonce) == 0 {
		t.Fatalf("expected non-empty ciphertext/nonce")
	}

	got, err := Decrypt(key, ciphertext, nonce, aad)
	if err != nil {
		t.Fatalf("Decrypt() err=%v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("roundtrip mismatch: got=%q want=%q", got, plaintext)
	}
}

func TestDecrypt_Errors(t *testing.T) {
	t.Parallel()

	key := bytes.Repeat([]byte{0x11}, 32)
	plaintext := []byte("hello")
	aad := []byte("aad")
	ciphertext, nonce, err := Encrypt(key, plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt() err=%v", err)
	}

	tests := []struct {
		name    string
		key     []byte
		ct      []byte
		nonce   []byte
		aad     []byte
		wantErr bool
	}{
		{
			name:    "bad_key_len",
			key:     []byte{1, 2, 3},
			ct:      ciphertext,
			nonce:   nonce,
			aad:     aad,
			wantErr: true,
		},
		{
			name:    "bad_nonce_len",
			key:     key,
			ct:      ciphertext,
			nonce:   []byte{1, 2, 3},
			aad:     aad,
			wantErr: true,
		},
		{
			name:    "aad_mismatch",
			key:     key,
			ct:      ciphertext,
			nonce:   nonce,
			aad:     []byte("other"),
			wantErr: true,
		},
		{
			name:    "ciphertext_corrupt",
			key:     key,
			ct:      append([]byte(nil), ciphertext[:len(ciphertext)-1]...),
			nonce:   nonce,
			aad:     aad,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Decrypt(tt.key, tt.ct, tt.nonce, tt.aad)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Decrypt() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

