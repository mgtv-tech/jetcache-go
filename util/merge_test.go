package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeMap(t *testing.T) {
	t.Run("merge one map", func(t *testing.T) {
		m1 := map[int]string{
			1: "a",
			2: "b",
		}
		expected := map[int]string{
			1: "a",
			2: "b",
		}
		actual := MergeMap(m1)
		assert.Equal(t, expected, actual)
	})

	t.Run("merge when map1 nil", func(t *testing.T) {
		var m1 map[int]string = nil
		m2 := map[int]string{
			1: "1",
			3: "2",
		}
		expected := map[int]string{
			1: "1",
			3: "2",
		}
		actual := MergeMap(m1, m2)
		assert.Equal(t, expected, actual)
	})

	t.Run("merge maps", func(t *testing.T) {
		m1 := map[int]string{
			1: "a",
			2: "b",
		}
		m2 := map[int]string{
			1: "1",
			3: "2",
		}

		expected := map[int]string{
			1: "1",
			2: "b",
			3: "2",
		}
		actual := MergeMap(m1, m2)

		assert.Equal(t, expected, actual)
	})
}
