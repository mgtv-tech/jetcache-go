package util

import (
	"bytes"
	"fmt"
	"sync"
)

var bfPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer([]byte{})
	},
}

func JoinAny(sep string, elems ...interface{}) string {
	if len(elems) == 0 {
		return ""
	}
	if len(elems) == 1 {
		return fmt.Sprintf("%v", elems[0])
	}
	buf := bfPool.Get().(*bytes.Buffer)
	buf.WriteString(fmt.Sprintf("%v", elems[0]))
	for i := 1; i < len(elems); i++ {
		buf.WriteString(sep)
		buf.WriteString(fmt.Sprintf("%v", elems[i]))
	}
	s := buf.String()
	buf.Reset()
	bfPool.Put(buf)
	return s
}
