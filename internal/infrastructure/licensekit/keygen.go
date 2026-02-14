package licensekit

import (
	"crypto/rand"
	"fmt"
)

// GenerateLicenseKey 生成 license key
// 格式：MDF-XXXX-XXXX-XXXX（全随机，避免被猜测）
func GenerateLicenseKey() string {
	// 生成三个随机部分，每部分 4 个字符
	part1 := generateRandomString(4)
	part2 := generateRandomString(4)
	part3 := generateRandomString(4)

	return fmt.Sprintf("MDF-%s-%s-%s", part1, part2, part3)
}

// generateRandomString 生成随机字符串
// 使用加密安全的随机数生成器
func generateRandomString(length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 排除易混淆字符 0,O,1,I
	b := make([]byte, length)
	rand.Read(b)

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[int(b[i])%len(charset)]
	}

	return string(result)
}

