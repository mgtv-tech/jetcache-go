package local

type Local interface {
	// Set stores the given data with the specified key.
	Set(key string, data []byte)

	// Get retrieves the data associated with the specified key.
	// It returns the data and a boolean indicating whether the key was found.
	Get(key string) ([]byte, bool)

	// Del deletes the data associated with the specified key.
	Del(key string)
}
