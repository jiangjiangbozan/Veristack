package pack

import (
	"crypto/sha256"
	"encoding/hex"
)

// sha256Hex 计算内容的十六进制 SHA-256 摘要。
func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
