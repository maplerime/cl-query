/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: jwt operations
 *
**/

package common

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"math/rand"

	jwt "github.com/golang-jwt/jwt/v4"
)

const (
	DefaultExpiresAt = 2 * time.Hour
)

var (
	_publicKey  *rsa.PublicKey
	_privateKey *rsa.PrivateKey
)

type CustomClaims struct {
	jwt.RegisteredClaims
	UID string `json:"uid,omitempty"`
	OID string `json:"oid,omitempty"`
}

func (*CustomClaims) verifyPrivilege(resource interface{}) (result bool) {
	// TODO:  checkout authority
	return true
}

func NewClaims(u, o, uid, oid string) (claims jwt.Claims, issuedAt, ExpiresAt int64) {
	now := time.Now()
	issuedAt = now.Unix()
	if Config.Token.ExpiresAt == 0 {
		Config.Token.ExpiresAt = DefaultExpiresAt
	}
	ExpiresAt = now.Add(Config.Token.ExpiresAt).Unix()
	claims = &CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{u},
			ExpiresAt: jwt.NewNumericDate(time.Unix(ExpiresAt, 0)),
			ID:        claimsID(now),
			IssuedAt:  jwt.NewNumericDate(time.Unix(issuedAt, 0)),
			Issuer:    "Cloudland",
			NotBefore: jwt.NewNumericDate(time.Unix(issuedAt, 0)),
			Subject:   o,
		},
		UID: uid,
		OID: oid,
	}

	return
}

func claimsID(now time.Time) string {
	return fmt.Sprintf("%d", now.UnixNano()+rand.Int63())
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func publicKey() *rsa.PublicKey {
	if _publicKey == nil {
		keyFile := Config.Token.PublicKey
		if keyFile == "" {
			panic("No public key provided")
		}
		key, err := os.ReadFile(keyFile)
		if err != nil {
			panic(err)
		}
		if len(key) == 0 {
			panic("No public key provided")
		}
		_publicKey, err = jwt.ParseRSAPublicKeyFromPEM(key)
		if err != nil {
			panic(err)
		}
	}
	return _publicKey
}

func ParseToken(tokenString string) (token *jwt.Token, tokenClaims *CustomClaims, err error) {
	tokenClaims = &CustomClaims{}
	token, err = jwt.ParseWithClaims(
		tokenString,
		tokenClaims,
		func(token *jwt.Token) (interface{}, error) {
			return publicKey(), nil
		},
	)
	return
}
