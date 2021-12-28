package cache

var caches = make(map[string][]byte)

func Get(key string) ([]byte, error) {
	return caches[key], nil
}

func Set(key string, value []byte) error {
	caches[key] = value
	return nil
}
