package sign

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
)

func Generate() (*rsa.PrivateKey, error) {
	// The GenerateKey method takes in a reader that returns random bits, and
	// the number of bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrap(err, "generate key")
	}

	return privateKey, nil
}

func Marshal(key *rsa.PrivateKey) (privateKey, publicKey string) {
	privateKeyData := x509.MarshalPKCS1PrivateKey(key)
	publicKeyData := x509.MarshalPKCS1PublicKey(&key.PublicKey)

	return base64.StdEncoding.EncodeToString(privateKeyData), base64.StdEncoding.EncodeToString(publicKeyData)
}

func UnmarshalPrivate(privateKey string) (*rsa.PrivateKey, error) {
	uDec, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "decode string")
	}
	rsaPrivateKey, err := x509.ParsePKCS1PrivateKey(uDec)
	if err != nil {
		return nil, errors.Wrap(err, "parse PKCS1 private key")
	}

	return rsaPrivateKey, nil
}

func UnmarshalPublic(publicKey string) (*rsa.PublicKey, error) {
	uDec, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "decode string")
	}

	rsaPublicKey, err := x509.ParsePKCS1PublicKey(uDec)
	if err != nil {
		return nil, errors.Wrap(err, "parse PKCS1 public key")
	}

	return rsaPublicKey, nil
}

func DataSign(data interface{}, key *rsa.PrivateKey) (string, error) {
	d, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "marshal")
	}

	dataHash := sha256.New()
	if _, err = dataHash.Write(d); err != nil {
		return "", errors.Wrap(err, "write")
	}

	dataHashSum := dataHash.Sum(nil)

	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, dataHashSum, nil)
	if err != nil {
		return "", errors.Wrap(err, "sign PSS")
	}

	signString := base64.StdEncoding.EncodeToString(signature)
	return signString, nil
}
