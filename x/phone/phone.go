package phone

import (
	"strings"
)

// Normalize normilizes phone
func Normilize(phone string) (string, error) {
	normEmail := strings.TrimSpace(phone)
	normEmail = strings.TrimPrefix(normEmail, "+")
	normEmail = strings.ReplaceAll(normEmail, "-", "")
	normEmail = strings.ReplaceAll(normEmail, " ", "")
	return normEmail, nil
}
