package util

// MergeMap merge maps, next key will overwrite previous key.
func MergeMap[K comparable, V any](maps ...map[K]V) map[K]V {
	if len(maps) == 0 {
		return nil
	}

	if len(maps) == 1 {
		return maps[0]
	}

	result := maps[0]
	if result == nil {
		result = make(map[K]V)
	}

	for _, m := range maps[1:] {
		for k, v := range m {
			result[k] = v
		}
	}

	return result
}
