package model

import "crypto/rand"

type Id []byte

func NewId() Id {
	ret := make(Id, 32)
	if _, err := rand.Read(ret); err != nil {
		panic(err)
	}
	return ret
}