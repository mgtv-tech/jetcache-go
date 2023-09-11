package stats

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testHandler struct {
	Hit        uint64
	Miss       uint64
	LocalHit   uint64
	LocalMiss  uint64
	RemoteHit  uint64
	RemoteMiss uint64
	Query      uint64
	QueryFail  uint64
}

func TestNewHandles(t *testing.T) {
	tests := []struct {
		input  bool
		expect uint64
	}{
		{
			input:  false,
			expect: 1,
		},
		{
			input:  true,
			expect: 0,
		},
	}
	for _, v := range tests {
		var handler testHandler
		h := NewHandles(v.input, &handler)
		h.IncrHit()
		h.IncrMiss()
		h.IncrLocalHit()
		h.IncrLocalMiss()
		h.IncrRemoteHit()
		h.IncrRemoteMiss()
		h.IncrQuery()
		h.IncrQueryFail(errors.New("any"))

		assert.Equal(t, v.expect, handler.Hit)
		assert.Equal(t, v.expect, handler.Miss)
		assert.Equal(t, v.expect, handler.LocalHit)
		assert.Equal(t, v.expect, handler.LocalMiss)
		assert.Equal(t, v.expect, handler.RemoteHit)
		assert.Equal(t, v.expect, handler.RemoteMiss)
		assert.Equal(t, v.expect, handler.Query)
		assert.Equal(t, v.expect, handler.QueryFail)
	}
}

func (h *testHandler) IncrHit() {
	atomic.AddUint64(&h.Hit, 1)
}

func (h *testHandler) IncrMiss() {
	atomic.AddUint64(&h.Miss, 1)
}

func (h *testHandler) IncrLocalHit() {
	atomic.AddUint64(&h.LocalHit, 1)
}

func (h *testHandler) IncrLocalMiss() {
	atomic.AddUint64(&h.LocalMiss, 1)
}

func (h *testHandler) IncrRemoteHit() {
	atomic.AddUint64(&h.RemoteHit, 1)
}

func (h *testHandler) IncrRemoteMiss() {
	atomic.AddUint64(&h.RemoteMiss, 1)
}

func (h *testHandler) IncrQuery() {
	atomic.AddUint64(&h.Query, 1)
}

func (h *testHandler) IncrQueryFail(err error) {
	atomic.AddUint64(&h.QueryFail, 1)
}
