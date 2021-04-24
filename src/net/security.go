package net

import (
	"crypto/rsa"
)

type AuthHandler interface {
	Filter()
	Sign()
}

type NaiveAuthHandler struct {
	publicKey  rsa.PublicKey
	privateKey rsa.PrivateKey
}

func NewNaiveAuthHandler(pub rsa.PublicKey, pri rsa.PrivateKey) *NaiveAuthHandler {
	return &NaiveAuthHandler{
		publicKey:  pub,
		privateKey: pri,
	}
}

func (nah *NaiveAuthHandler) Filter() {

}

func (nah *NaiveAuthHandler) Sign() {
	// rsa.SignPKCS1v15()
}
