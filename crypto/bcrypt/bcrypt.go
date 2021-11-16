package bcrypt

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultCost = 8
)

func HashAndSalt(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), defaultCost)
	if err != nil {
		return "", errors.Wrap(err, "generate from password")
	}
	// GenerateFromPassword returns a byte slice so we need to
	// convert the bytes to a string and return it
	return string(hash), nil
}

func ComparePasswords(hashedPwd, plainPwd string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(plainPwd)); err != nil {
		return errors.Wrap(err, "compare hash and password")
	}

	return nil
}
