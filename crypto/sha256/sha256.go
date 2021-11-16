package sha256

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/crypto/random"
)

const (
	cryptAlg    = "sha256"
	cryptAlgLen = len(cryptAlg)
	defaultCost = 8
)

var (
	// ErrMismatchedHashAndPassword the error returned from CompareHashAndPassword when a password and hash do
	// not match.
	ErrMismatchedHashAndPassword = errors.New("hashedPassword is not the hash of the given password")
	// ErrBadAlgorithm the error returned from newFromHash when a hash not include necessary prefix
	ErrBadAlgorithm = errors.New("bad algorithm")
)

type hashed struct {
	hash []byte
	salt []byte
}

func bcrypt(password, salt []byte) ([]byte, error) {
	// Combine salt with password
	h := sha256.New()
	if _, err := h.Write(password); err != nil {
		return nil, errors.Wrap(err, "write password")
	}

	if _, err := h.Write(salt); err != nil {
		return nil, errors.Wrap(err, "write salt")
	}

	return []byte(hex.EncodeToString(h.Sum(nil))), nil
}

func newFromPassword(password []byte, saltSize int) (*hashed, error) {
	p := &hashed{}

	salt, err := random.GenerateSalt(saltSize)
	if err != nil {
		return nil, errors.Wrap(err, "generate salt")
	}
	p.hash, err = bcrypt(password, salt)
	if err != nil {
		return nil, errors.Wrap(err, "bcrypt")
	}
	p.salt = salt

	return p, nil
}

// GenerateFromPassword returns the sha1 hash of the password at the given
// salt.
func GenerateFromPassword(password []byte, saltSize int) ([]byte, error) {
	p, err := newFromPassword(password, saltSize)
	if err != nil {
		return nil, errors.Wrap(err, "new hashed from password")
	}
	return p.Hash(), nil
}

// Hash return hash-bytes in the format algorithm$salt$iterations$hash
func (p *hashed) Hash() []byte {
	arr := make([]byte, 0, cryptAlgLen+len(p.salt)+len(p.hash)+5)
	arr = append(arr, []byte(cryptAlg)...)
	arr = append(arr, '$')
	arr = append(arr, p.salt...)
	arr = append(arr, []byte("$1$")...) // iterations, always 1
	arr = append(arr, p.hash...)

	return arr
}

func CompareHashAndPassword(hashedPassword, password []byte, saltSize int) error {
	p, err := newFromHash(hashedPassword, saltSize)
	if err != nil {
		return errors.Wrap(err, "new hashed from hash+salt")
	}

	h, err := bcrypt(password, p.salt)
	if err != nil {
		return errors.Wrap(err, "bcrypt")
	}

	otherP := &hashed{
		hash: h,
		salt: p.salt,
	}

	if subtle.ConstantTimeCompare(p.Hash(), otherP.Hash()) != 1 {
		return ErrMismatchedHashAndPassword
	}

	return nil
}

func newFromHash(hashedPassword []byte, saltSize int) (*hashed, error) {
	p := &hashed{}
	// Check algorithm
	n := cryptAlgLen
	res := bytes.Compare(hashedPassword[:n], []byte(cryptAlg))

	if res != 0 {
		return nil, ErrBadAlgorithm
	}

	// Salt field
	n++
	p.salt = hashedPassword[n : n+saltSize]

	// Skip iterations field
	n += saltSize
	n += 3

	// Hash field
	p.hash = hashedPassword[n:]

	return p, nil
}

func HashAndSalt(pwd string) (string, error) {
	hash, err := GenerateFromPassword([]byte(pwd), defaultCost)
	if err != nil {
		return "", errors.Wrap(err, "generate from password")
	}
	// GenerateFromPassword returns a byte slice so we need to
	// convert the bytes to a string and return it
	return string(hash), nil
}

func ComparePasswords(hashedPwd, plainPwd string) error {
	return CompareHashAndPassword([]byte(hashedPwd), []byte(plainPwd), defaultCost)
}
