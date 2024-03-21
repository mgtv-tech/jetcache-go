package sonic

import (
	"github.com/bytedance/sonic"
	"github.com/mgtv-tech/jetcache-go/encoding"
)

// Name is the name registered for the json codec.
const Name = "sonic"

func init() {
	encoding.RegisterCodec(codec{})
}

// codec is a Codec implementation with json.
type codec struct{}

func (codec) Marshal(v interface{}) ([]byte, error) {
	return sonic.Marshal(v)
}

func (codec) Unmarshal(data []byte, v interface{}) error {
	return sonic.Unmarshal(data, v)
}

func (codec) Name() string {
	return Name
}
