package token

import (
	"crypto/rsa"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// Verifier validates RS256 access tokens with the public key only. This service
// never signs tokens — only auth-service holds the private key.
type Verifier struct {
	publicKey *rsa.PublicKey
	issuer    string
}

// Claims is the JWT payload. Role is duplicated into the token so downstream
// services can authorize without calling back to Auth.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// NewVerifier loads the RSA public key from disk and returns a verifier bound to
// the expected issuer.
func NewVerifier(pubPath, issuer string) (*Verifier, error) {
	pubPEM, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, fmt.Errorf("read public key %q: %w", pubPath, err)
	}
	pub, err := jwt.ParseRSAPublicKeyFromPEM(pubPEM)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return &Verifier{publicKey: pub, issuer: issuer}, nil
}

// Verify validates the signature (RS256 only), issuer and expiry, returning the
// claims.
func (v *Verifier) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.publicKey, nil
	}, jwt.WithIssuer(v.issuer), jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}
