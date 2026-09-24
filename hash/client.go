package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Client is the redirect log's Client field (#19): HMAC-SHA256 of the client
// address and User-Agent under the quarter's salt, cut to 64 bits. That is
// plenty to count distinct clients in a quarter, and without the salt it
// cannot be traced back to an address or matched across quarters.
func Client(salt []byte, ip, userAgent string) string {
	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(ip))
	mac.Write([]byte{0})
	mac.Write([]byte(userAgent))
	return hex.EncodeToString(mac.Sum(nil)[:8])
}
