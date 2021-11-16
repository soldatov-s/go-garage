package stringsx

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// JoinStrings works as strings.Join, but can receive arbitrary number of strings
// The separator string sep is placed between elements in the resulting string.
func JoinStrings(sep string, elems ...string) string {
	return strings.Join(elems, sep)
}

func RedactedDSN(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", errors.Wrap(err, "parse dsn")
	}

	return u.Redacted(), nil
}

func ReverseStringSlice(source []string) []string {
	dest := make([]string, 0, len(source))
	for i := len(source) - 1; i >= 0; i-- {
		dest = append(dest, source[i])
	}
	return dest
}
