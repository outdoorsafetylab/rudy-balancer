package hash

import "testing"

func TestClient(t *testing.T) {
	salt := []byte("0123456789abcdef0123456789abcdef")
	id := Client(salt, "203.0.113.7", "wadi/7.03")
	// Known answer from Python: hmac.new(salt, b"203.0.113.7\x00wadi/7.03",
	// hashlib.sha256).hexdigest()[:16]
	if id != "3a7f74a91601f8eb" {
		t.Fatalf("got %q, want HMAC-SHA256 known answer 3a7f74a91601f8eb", id)
	}
	if again := Client(salt, "203.0.113.7", "wadi/7.03"); again != id {
		t.Errorf("same input gave %q then %q", id, again)
	}
	for name, other := range map[string]string{
		"other salt": Client([]byte("another salt"), "203.0.113.7", "wadi/7.03"),
		"other ip":   Client(salt, "203.0.113.8", "wadi/7.03"),
		"other ua":   Client(salt, "203.0.113.7", "wadi/7.04"),
		// The separator keeps "ab"+"c" apart from "a"+"bc".
		"shifted": Client(salt, "203.0.113.7w", "adi/7.03"),
	} {
		if other == id {
			t.Errorf("%s: collided with %q", name, id)
		}
	}
}
