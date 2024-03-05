package stats

type (
	// Handler defines the interface that the Transport uses to collect cache metrics.
	// Note that implementations of this interface must be thread-safe; the methods of a Handler
	// can be called from concurrent goroutines.
	Handler interface {
		IncrHit()
		IncrMiss()
		IncrLocalHit()
		IncrLocalMiss()
		IncrRemoteHit()
		IncrRemoteMiss()
		IncrQuery()
		IncrQueryFail(err error)
	}

	Handlers struct {
		disable  bool
		handlers []Handler
	}
)

// NewHandles creates a new instance of Handlers.
func NewHandles(disable bool, handlers ...Handler) Handler {
	return &Handlers{
		disable:  disable,
		handlers: handlers,
	}
}

func (hs *Handlers) IncrHit() {
	if hs.disable {
		return
	}

	for _, h := range hs.handlers {
		h.IncrHit()
	}
}

func (hs *Handlers) IncrMiss() {
	if hs.disable {
		return
	}

	for _, h := range hs.handlers {
		h.IncrMiss()
	}
}

func (hs *Handlers) IncrLocalHit() {
	if hs.disable {
		return
	}

	for _, h := range hs.handlers {
		h.IncrLocalHit()
	}
}

func (hs *Handlers) IncrLocalMiss() {
	if hs.disable {
		return
	}

	for _, h := range hs.handlers {
		h.IncrLocalMiss()
	}
}

func (hs *Handlers) IncrRemoteHit() {
	if hs.disable {
		return
	}

	for _, h := range hs.handlers {
		h.IncrRemoteHit()
	}
}

func (hs *Handlers) IncrRemoteMiss() {
	if hs.disable {
		return
	}

	for _, h := range hs.handlers {
		h.IncrRemoteMiss()
	}
}

func (hs *Handlers) IncrQuery() {
	if hs.disable {
		return
	}

	for _, h := range hs.handlers {
		h.IncrQuery()
	}
}

func (hs *Handlers) IncrQueryFail(err error) {
	if hs.disable {
		return
	}

	for _, h := range hs.handlers {
		h.IncrQueryFail(err)
	}
}
