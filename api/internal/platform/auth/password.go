// Package auth 的密码文件，负责密码哈希与校验。
package auth

import "golang.org/x/crypto/bcrypt"

const hashCost = 12

// HashPassword 将明文密码哈希为不可逆摘要，签发前不做截断以保留完整熵。
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword 校验明文密码与哈希是否匹配。
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
