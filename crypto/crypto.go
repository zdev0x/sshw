package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// 加密参数
	saltSize   = 16
	keySize    = 32 // AES-256
	iterations = 100000
	nonceSize  = 12
	tagSize    = 16
)

// Encrypt 使用 AES-256-GCM 加密数据
func Encrypt(plaintext []byte, key []byte) (string, error) {
	// 生成随机盐值
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// 使用 PBKDF2 派生密钥
	derivedKey := pbkdf2.Key(key, salt, iterations, keySize, sha256.New)

	// 创建 AES 密码块
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// 组合盐值和密文
	result := make([]byte, 0, len(salt)+len(ciphertext))
	result = append(result, salt...)
	result = append(result, ciphertext...)

	// Base64 编码
	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt 解密 AES-256-GCM 加密的数据
func Decrypt(encrypted string, key []byte) ([]byte, error) {
	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	if len(data) < saltSize+nonceSize+tagSize {
		return nil, errors.New("encrypted data too short")
	}

	// 提取盐值和密文
	salt := data[:saltSize]
	ciphertext := data[saltSize:]

	// 使用 PBKDF2 派生密钥
	derivedKey := pbkdf2.Key(key, salt, iterations, keySize, sha256.New)

	// 创建 AES 密码块
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 提取 nonce
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	// 解密数据
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// IsEncrypted 检查字符串是否是加密格式
func IsEncrypted(s string) bool {
	// 检查是否是有效的 Base64 编码
	_, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return false
	}
	return true
}
