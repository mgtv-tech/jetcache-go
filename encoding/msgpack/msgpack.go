package msgpack

import (
	"fmt"

	"github.com/klauspost/compress/s2"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/mgtv-tech/jetcache-go/encoding"
)

// Name is the name registered for the json codec.
const (
	Name = "msgpack"

	compressionThreshold = 64
	timeLen              = 4

	noCompression = 0x0
	s2Compression = 0x1
)

func init() {
	encoding.RegisterCodec(codec{})
}

// codec is a Codec implementation with json.
type codec struct{}

func (codec) Marshal(v any) ([]byte, error) {
	b, err := msgpack.Marshal(v)
	if err != nil {
		return nil, err
	}

	return compress(b), nil
}

func (codec) Unmarshal(data []byte, v any) error {
	switch c := data[len(data)-1]; c {
	case noCompression:
		data = data[:len(data)-1]
	case s2Compression:
		data = data[:len(data)-1]

		var err error
		data, err = s2.Decode(nil, data)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown compression method: %x", c)
	}

	return msgpack.Unmarshal(data, v)
}

func (codec) Name() string {
	return Name
}

func compress(data []byte) []byte {
	if len(data) < compressionThreshold {
		n := len(data) + 1
		b := make([]byte, n, n+timeLen)
		copy(b, data)
		b[len(b)-1] = noCompression
		return b
	}

	n := s2.MaxEncodedLen(len(data)) + 1
	b := make([]byte, n, n+timeLen)
	b = s2.Encode(b, data)
	b = append(b, s2Compression)
	return b
}
