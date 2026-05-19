package executor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type RcloneConfig struct {
	Type   string
	Params map[string]string
}

func WriteRcloneConf(path string, config RcloneConfig) error {
	var builder strings.Builder
	builder.WriteString("[vaultfleet]\n")
	builder.WriteString(fmt.Sprintf("type = %s\n", config.Type))

	keys := make([]string, 0, len(config.Params))
	for key := range config.Params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value, err := rcloneConfigValue(config.Type, key, config.Params[key])
		if err != nil {
			return err
		}
		builder.WriteString(fmt.Sprintf("%s = %s\n", key, value))
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := file.Chmod(0o600); err != nil {
		return err
	}
	if _, err := file.WriteString(builder.String()); err != nil {
		return err
	}
	return nil
}

func rcloneConfigValue(configType string, key string, value string) (string, error) {
	if configType == "webdav" && key == "pass" && value != "" {
		return obscureRcloneValue(value)
	}
	return value, nil
}

var rcloneObscureKey = []byte{
	0x9c, 0x93, 0x5b, 0x48, 0x73, 0x0a, 0x55, 0x4d,
	0x6b, 0xfd, 0x7c, 0x63, 0xc8, 0x86, 0xa9, 0x2b,
	0xd3, 0x90, 0x19, 0x8e, 0xb8, 0x12, 0x8a, 0xfb,
	0xf4, 0xde, 0x16, 0x2b, 0x8b, 0x95, 0xf6, 0x38,
}

func obscureRcloneValue(value string) (string, error) {
	plaintext := []byte(value)
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("generate rclone obscure iv: %w", err)
	}
	if err := cryptRcloneValue(ciphertext[aes.BlockSize:], plaintext, iv); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func revealRcloneObscured(value string) (string, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed when revealing password - is it obscured?: %w", err)
	}
	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("input too short when revealing password - is it obscured?")
	}
	iv := ciphertext[:aes.BlockSize]
	buf := ciphertext[aes.BlockSize:]
	if err := cryptRcloneValue(buf, buf, iv); err != nil {
		return "", err
	}
	return string(buf), nil
}

func cryptRcloneValue(out []byte, in []byte, iv []byte) error {
	block, err := aes.NewCipher(rcloneObscureKey)
	if err != nil {
		return fmt.Errorf("create rclone obscure cipher: %w", err)
	}
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(out, in)
	return nil
}
