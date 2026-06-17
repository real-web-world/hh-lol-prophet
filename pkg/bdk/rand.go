package bdk

import (
	cryproRand "crypto/rand"
	"encoding/binary"
	"math/rand/v2"
	"time"
)

func NewCryptoRand() *rand.Rand {
	buf := make([]byte, 64)
	_, _ = cryproRand.Reader.Read(buf)
	seed1 := binary.LittleEndian.Uint64(buf)
	seed2 := uint64(time.Now().UnixNano())
	return rand.New(rand.NewPCG(seed1, seed2))
}
