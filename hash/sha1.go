package hash

import (
	"crypto/sha1"
	"encoding/hex"
)

func SHA1(data []byte) (string, error) {
	h := sha1.New()
	if _, err := h.Write(data); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	return sum, nil
}
