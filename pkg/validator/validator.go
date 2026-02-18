package validator

import (
	"regexp"
	"strings"
)

// ValidatePhone 验证手机号
func ValidatePhone(phone string) bool {
	if phone == "" {
		return false
	}

	// 简单的手机号验证（中国）
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, phone)
	return matched
}

// ValidateEmail 验证邮箱
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidateNickname 验证昵称
func ValidateNickname(nickname string) bool {
	if nickname == "" {
		return false
	}

	// 1-20个字符
	if len([]rune(nickname)) < 1 || len([]rune(nickname)) > 20 {
		return false
	}

	return true
}

// ContainsSensitiveWords 检查是否包含敏感词
func ContainsSensitiveWords(text string) bool {
	// 简单的敏感词列表，实际应该从数据库或配置文件读取
	sensitiveWords := []string{"admin", "系统", "客服", "官方"}

	lowerText := strings.ToLower(text)
	for _, word := range sensitiveWords {
		if strings.Contains(lowerText, strings.ToLower(word)) {
			return true
		}
	}

	return false
}

// ValidateDeviceType 验证设备类型
func ValidateDeviceType(deviceType string) bool {
	validTypes := []string{"iOS", "Android", "Web", "PC"}
	for _, t := range validTypes {
		if deviceType == t {
			return true
		}
	}
	return false
}

// ValidateGender 验证性别
func ValidateGender(gender int) bool {
	return gender >= 0 && gender <= 2
}

// SanitizeString 清理字符串
func SanitizeString(str string) string {
	return strings.TrimSpace(str)
}
