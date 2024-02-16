package phone

import (
	"strings"
)

// Normalize normilizes phone
func Normilize(phone string) (string, error) {
	normPhone := strings.TrimSpace(phone)
	normPhone = strings.TrimPrefix(normPhone, "+")
	normPhone = strings.ReplaceAll(normPhone, "-", "")
	normPhone = strings.ReplaceAll(normPhone, " ", "")
	return normPhone, nil
}
