package common

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

/*
	# Private key & Public key
	Server Node Request Register data to decode
*/

var (
	ErrDecodePriKey = errors.New("decode private key string error")
	ErrParsePriKey  = errors.New("can not parse private key type")
	ErrAssertionPri = errors.New("could not parse key to *rsa.PrivateKey")

	ErrDecodePubKey = errors.New("decode public key string error")
	ErrAssertionPub = errors.New("could not parse key to *rsa.PublickKey")
)

// ParsePrivateKey : parse private key bytes to rsa.PrivateKey
func ParsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, ErrDecodePriKey
	}

	var key interface{}
	var err error

	if block.Type == "RSA PRIVATE KEY" {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	} else if block.Type == "PRIVATE KEY" {
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	} else {
		return nil, ErrParsePriKey
	}

	if err != nil {
		return nil, err
	}

	if pri, ok := key.(*rsa.PrivateKey); ok {
		return pri, nil
	}
	return nil, ErrAssertionPri
}

// ParsePublicKey : parse public key bytes to rsa.PublicKey
func ParsePublicKey(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, ErrDecodePubKey
	}

	if block.Type == "CERTIFICATE" {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			return cert.PublicKey.(*rsa.PublicKey), nil
		}
	} else if block.Type == "PUBLIC KEY" {
		if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
			if pub, ok := key.(*rsa.PublicKey); ok {
				return pub, nil
			}
		}
	}
	return nil, ErrAssertionPub
}

// SignRSA : Sign map to SHA256
func SignRSAData(data []byte, priKey *rsa.PrivateKey) ([]byte, error) {
	hash := crypto.SHA256
	h := hash.New()
	h.Write(data)
	hashed := h.Sum(nil)
	return rsa.SignPKCS1v15(rand.Reader, priKey, hash, hashed)
}

// VerifySignRSA : Verify map data with SHA256
func VerifySignRSA(data []byte, pubKey *rsa.PublicKey) error {
	hash := crypto.SHA256
	h := hash.New()
	h.Write(data)
	hashed := h.Sum(nil)
	return rsa.VerifyPKCS1v15(pubKey, hash, hashed, data)
}
