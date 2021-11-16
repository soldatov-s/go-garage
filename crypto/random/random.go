package random

import (
	"crypto/rand"
	"math/big"

	"github.com/pkg/errors"
)

type RuneSequenceType int

const (
	// AlphaNum contains runes [abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789].
	AlphaNum RuneSequenceType = iota
	// Alpha contains runes [abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ].
	Alpha
	// AlphaLowerNum contains runes [abcdefghijklmnopqrstuvwxyz0123456789].
	AlphaLowerNum
	// AlphaUpperNum contains runes [ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789].
	AlphaUpperNum
	// AlphaLower contains runes [abcdefghijklmnopqrstuvwxyz].
	AlphaLower
	// AlphaUpper contains runes [ABCDEFGHIJKLMNOPQRSTUVWXYZ].
	AlphaUpper
	// Numeric contains runes [0123456789].
	Numeric
)

type RuneArray []rune

func (r RuneSequenceType) Runes() []rune {
	return []RuneArray{
		[]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"),
		[]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		[]rune("abcdefghijklmnopqrstuvwxyz0123456789"),
		[]rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"),
		[]rune("abcdefghijklmnopqrstuvwxyz"),
		[]rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		[]rune("0123456789"),
	}[r]
}

// Random returns a random sequence using the defined allowed runes.
func (r RuneSequenceType) Random(l int) (seq RuneArray, err error) {
	rander := rand.Reader

	c := big.NewInt(int64(len(r.Runes())))
	seq = make([]rune, l)

	for i := 0; i < l; i++ {
		rnd, err := rand.Int(rander, c)
		if err != nil {
			return seq, errors.Wrap(err, "random int")
		}
		rn := r.Runes()[rnd.Uint64()]
		seq[i] = rn
	}

	return seq, nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func GenerateSalt(saltSize int) ([]byte, error) {
	l := len(letterBytes)
	b := make([]byte, saltSize)
	for i := range b {
		id, err := rand.Int(rand.Reader, big.NewInt(int64(l)))
		if err != nil {
			return nil, errors.Wrap(err, "random int")
		}
		b[i] = letterBytes[id.Int64()]
	}
	return b, nil
}
