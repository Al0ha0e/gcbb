package common

import "crypto/sha1"

func GenSHA1(data []byte) HashVal {
	return sha1.Sum(data)
}
