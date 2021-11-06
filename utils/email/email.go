package email

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	GMAIL      = "gmail.com"
	GOOGLEMAIL = "googlemail.com"
	OUTLOOK    = "outlook.com"
	ICLOUD     = "icloud.com"
	YAHOO      = "yahoo.com"
)

// Parse email
func Parse(email string) (user, domain string, err error) {
	// validate email
	strs := strings.Split(email, "@")
	if len(strs) < 2 {
		return "", "", ErrNormilizeEmail
	}

	user = strs[0]
	domain = strs[1]
	return user, domain, nil
}

// Normalize normilizes email address to form name@domain
func Normilize(email string) (string, error) {
	// all_lowercase
	normEmail := strings.ToLower(email)

	user, domain, err := Parse(normEmail)
	if err != nil {
		return "", errors.Wrap(err, "parse email")
	}

	// Converts addresses with domain @googlemail.com to @gmail.com, as they're equivalent.
	domain = strings.ReplaceAll(domain, GOOGLEMAIL, GMAIL)

	switch domain {
	case GMAIL:
		// Removes dots from the local part of the email address, as GMail ignores them
		// (e.g. "john.doe" and "johndoe" are considered equal).
		user = strings.ReplaceAll(user, ".", "")
		fallthrough
	case OUTLOOK, ICLOUD:
		// Normalizes addresses by removing "sub-addresses", which is the part following a "+" sign
		// (e.g. "foo+bar@gmail.com" becomes "foo@gmail.com", "foo+bar@outlook.com" becomes "foo@outlook.com",
		// "foo+bar@icloud.com" becomes "foo@icloud.com".
		if id := strings.Index(user, "+"); id > 0 {
			user = user[0:id]
		}
	case YAHOO:
		// Normalizes addresses by removing "sub-addresses", which is the part following a "-"
		// sign (e.g. "foo-bar@yahoo.com" becomes "foo@yahoo.com").
		if id := strings.Index(user, "-"); id > 0 {
			user = user[0:id]
		}
	}

	return user + "@" + domain, nil
}
