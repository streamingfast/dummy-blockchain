package core

import (
	"crypto/sha256"
	"fmt"
)

func makeHash(data interface{}) string {
	shaSum := sha256.Sum256([]byte(fmt.Sprintf("%v", data)))
	return fmt.Sprintf("%x", shaSum)
}
