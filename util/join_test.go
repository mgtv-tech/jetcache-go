package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type object struct {
	Str string
	Num int
}

func TestJoinAny(t *testing.T) {
	assert.Equal(t, "", JoinAny(","))
	assert.Equal(t, "", JoinAny(",", ""))
	assert.Equal(t, "<nil>", JoinAny(",", nil))
	assert.Equal(t, "a", JoinAny(",", "a"))
	assert.Equal(t, "a,b,c", JoinAny(",", "a", "b", "c"))
	assert.Equal(t, "a,1,0.3", JoinAny(",", "a", 1, 0.3))
	assert.Equal(t, "a,[1 2 3]", JoinAny(",", "a", []int{1, 2, 3}))
	assert.Equal(t, "a,&{join 42}", JoinAny(",", "a", &object{Str: "join", Num: 42}))

}
