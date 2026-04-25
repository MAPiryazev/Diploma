package security

import "strings"

func MaskID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return "****"
	}
	return id[:4] + "..." + id[len(id)-4:]
}

func MaskAmount(amount string) string {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return "***"
	}
	return "***"
}
