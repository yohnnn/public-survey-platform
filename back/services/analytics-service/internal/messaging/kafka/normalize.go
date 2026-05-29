package kafka

import "strings"

func normalizeGender(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "m", "male", "man", "мужской", "мужчина":
		return "male"
	case "f", "female", "woman", "женский", "женщина":
		return "female"
	default:
		return ""
	}
}
