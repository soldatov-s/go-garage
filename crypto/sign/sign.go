package sign

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
)

func Generate() (*rsa.PrivateKey, error) {
	// The GenerateKey method takes in a reader that returns random bits, and
	// the number of bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func Marshal(key *rsa.PrivateKey) (privateKey, publicKey string) {
	privateKeyData := x509.MarshalPKCS1PrivateKey(key)
	publicKeyData := x509.MarshalPKCS1PublicKey(&key.PublicKey)

	return base64.StdEncoding.EncodeToString(privateKeyData), base64.StdEncoding.EncodeToString(publicKeyData)
}

func UnmarshalPrivate(privateKey string) (*rsa.PrivateKey, error) {
	uDec, _ := base64.StdEncoding.DecodeString(privateKey)
	return x509.ParsePKCS1PrivateKey(uDec)
}

func UnmarshalPublic(publicKey string) (*rsa.PublicKey, error) {
	uDec, _ := base64.StdEncoding.DecodeString(publicKey)
	return x509.ParsePKCS1PublicKey(uDec)
}

func DataSign(data interface{}, key *rsa.PrivateKey) (string, error) {
	d, err := json.Marshal(data)

	if err != nil {
		return "", err
	}

	dataHash := sha256.New()
	_, err = dataHash.Write(d)
	if err != nil {
		return "", err
	}
	dataHashSum := dataHash.Sum(nil)

	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, dataHashSum, nil)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}
