package clientcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/makimaki04/go-data-keeper.git/cmd/pkg/contract"
	"golang.org/x/crypto/argon2"
)

func DeriveKey(password string, kdfSalt []byte, kdfParams contract.Params) ([]byte, error) {
	if len(kdfSalt) != 16 {
		return nil, fmt.Errorf("wrong derive key params")
	}

	if kdfParams.Parallelism > 255 {
		return nil, fmt.Errorf("wrong derive key params")
	}

	key := argon2.IDKey([]byte(password),
		kdfSalt,
		kdfParams.Time,
		kdfParams.Memory,
		uint8(kdfParams.Parallelism),
		kdfParams.KeyLen,
	)

	return key, nil
}

func Encrypt(key []byte, plaintext []byte, aad []byte) (ciphertext []byte, nonce []byte, err error) {
	if len(key) != 32 {
		return nil, nil, fmt.Errorf("wrong encrypt params: invalid key len %d", len(key))
	}

	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("aes.NewCipher error: %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, nil, fmt.Errorf("cipher.NewGCM error: %w", err)
	}

	nonce, err = generateRandom(aesgcm.NonceSize())
	if err != nil {
		return nil, nil, fmt.Errorf("generate nonce error: %w", err)
	}

	ciphertext = aesgcm.Seal(nil, nonce, plaintext, aad)

	return ciphertext, nonce, nil
}

func Decrypt(key []byte, ciphertext []byte, nonce []byte, aad []byte) (plaintext []byte, err error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("wrong decrypt params: invalid key len %d", len(key))
	}

	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher error: %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM error: %w", err)
	}

	if len(nonce) != aesgcm.NonceSize() {
		return nil, fmt.Errorf("wrong decrypt params: invalid nonce len %d", len(nonce))
	}

	plaintext, err = aesgcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("aesgcm.Open error: %w", err)
	}

	return plaintext, nil
}

func generateRandom(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
