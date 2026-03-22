package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptionKey — глобальный ключ шифрования, задаётся из переменной окружения при старте приложения.
var EncryptionKey string

// deriveKey приводит произвольную строку-ключ к 32 байтам (AES-256) через SHA-256.
func deriveKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}

// Encrypt шифрует plaintext с помощью AES-256-GCM и возвращает base64-строку вида nonce|ciphertext.
// Возвращает пустую строку без ошибки если plaintext пустой.
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if EncryptionKey == "" {
		return "", errors.New("encryption key is not set")
	}

	block, err := aes.NewCipher(deriveKey(EncryptionKey))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt расшифровывает строку, зашифрованную через Encrypt.
// Возвращает пустую строку без ошибки если ciphertext пустой.
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if EncryptionKey == "" {
		return "", errors.New("encryption key is not set")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// Если строка не является base64 — она, вероятно, ещё не зашифрована (legacy).
		// Возвращаем как есть, чтобы не сломать уже существующие данные.
		return ciphertext, nil
	}

	block, err := aes.NewCipher(deriveKey(EncryptionKey))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		// Данные слишком короткие — возможно, незашифрованные legacy-данные.
		return ciphertext, nil
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		// Расшифровка не удалась — возможно, legacy-данные. Возвращаем как есть.
		return ciphertext, nil
	}

	return string(plaintext), nil
}
