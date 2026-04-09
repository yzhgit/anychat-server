package validator

import (
	"regexp"
	"strings"
)

// ValidatePhone validates phone number
func ValidatePhone(phone string) bool {
	if phone == "" {
		return false
	}

	// Simple phone number validation (China)
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, phone)
	return matched
}

// ValidateEmail validates email
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidateNickname validates nickname
func ValidateNickname(nickname string) bool {
	if nickname == "" {
		return false
	}

	// 1-20 characters
	if len([]rune(nickname)) < 1 || len([]rune(nickname)) > 20 {
		return false
	}

	return true
}

// ContainsSensitiveWords checks if text contains sensitive words
func ContainsSensitiveWords(text string) bool {
	// Simple sensitive word list, should actually be read from database or config file
	sensitiveWords := []string{"admin", "system", "customer service", "official"}

	lowerText := strings.ToLower(text)
	for _, word := range sensitiveWords {
		if strings.Contains(lowerText, strings.ToLower(word)) {
			return true
		}
	}

	return false
}

// ValidateDeviceType validates device type
func ValidateDeviceType(deviceType string) bool {
	validTypes := []string{"iOS", "Android", "Web", "PC"}
	for _, t := range validTypes {
		if deviceType == t {
			return true
		}
	}
	return false
}

// ValidateGender validates gender
func ValidateGender(gender int) bool {
	return gender >= 0 && gender <= 2
}

// SanitizeString sanitizes string
func SanitizeString(str string) string {
	return strings.TrimSpace(str)
}
