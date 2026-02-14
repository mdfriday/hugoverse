package licensekit

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// LicenseKeyToEmail 将 License Key 转换为邮箱
// 规则：去掉 "MDF-" 前缀，转小写，加上 @mdfriday.com
// 例如：MDF-ABCD-EFGH-IJKL -> abcd-efgh-ijkl@mdfriday.com
func LicenseKeyToEmail(licenseKey string) string {
	key := strings.ToLower(strings.TrimPrefix(licenseKey, "MDF-"))
	return fmt.Sprintf("%s@mdfriday.com", key)
}

// LicenseKeyToPassword 将 License Key 转换为密码
// 规则：去掉 "MDF-" 前缀，转小写，base64 编码
func LicenseKeyToPassword(licenseKey string) string {
	key := strings.ToLower(strings.TrimPrefix(licenseKey, "MDF-"))
	return base64.StdEncoding.EncodeToString([]byte(key))
}

