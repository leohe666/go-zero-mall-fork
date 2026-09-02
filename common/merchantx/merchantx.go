// Package merchantx 商户凭据加解密（SaaS 多商户）。
//
// 安全模型：
//   - 每个商户的微信 AppSecret 不落配置文件/环境变量/git，只以 AES-GCM 密文存于
//     merchant 表的 wx_app_secret_enc 列；
//   - 唯一落地的密钥是"平台主密钥 MerchantMasterKey"（32 字节），属于部署级配置
//     （生产建议 KMS，开发可用 gitignore 的 key 文件），由 user rpc 持有；
//   - gateway 侧不接触主密钥：登录时经 GetMerchant RPC 拿到解密后的明文凭据。
package merchantx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Decrypt 解密 AES-GCM 密文。密文格式：base64(水滴IV(12) || ciphertext || tag(16))。
func Decrypt(masterKeyHex, encrypted string) (string, error) {
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid master key hex: %w", err)
	}
	if len(key) != 32 {
		return "", fmt.Errorf("master key must be 32 bytes (64 hex chars), got %d", len(key))
	}
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext error: %w", err)
	}
	if len(raw) < 12+16 {
		return "", errors.New("ciphertext too short")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce, ct := raw[:aead.NonceSize()], raw[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt error (wrong master key?): %w", err)
	}
	return string(plain), nil
}

// Encrypt 用 AES-GCM 加密明文，返回 base64(nonce+ct+tag)。
func Encrypt(masterKeyHex, plain string) (string, error) {
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid master key hex: %w", err)
	}
	if len(key) != 32 {
		return "", fmt.Errorf("master key must be 32 bytes (64 hex chars), got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := aead.Seal(nil, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ct...)), nil
}

// NewMasterKey 生成随机 32 字节主密钥（hex 编码），用于初始化/轮换。
func NewMasterKey() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// LoadMasterKey 从文件读取 hex 主密钥（自动 trim 空白）。
func LoadMasterKey(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read master key file: %w", err)
	}
	key := strings.TrimSpace(string(b))
	if len(key) != 64 {
		return "", fmt.Errorf("master key file must contain 64 hex chars, got %d", len(key))
	}
	return key, nil
}