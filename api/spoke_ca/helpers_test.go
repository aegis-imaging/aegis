package spoke_ca_test

import (
	"crypto/rand"
	"math/big"
	"time"
)

func mustSerial() *big.Int {
	max := new(big.Int).Lsh(big.NewInt(1), 64)
	n, _ := rand.Int(rand.Reader, max)
	return n
}

func timeNow() time.Time {
	return time.Now().UTC()
}
