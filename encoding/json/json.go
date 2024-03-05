package json

import (
	"encoding/json"

	"github.com/daoshenzzg/jetcache-go/encoding"
)

// Name is the name registered for the json codec.
const Name = "json"

func init() {
	encoding.RegisterCodec(codec{})
}

// codec is a Codec implementation with json.
type codec struct{}

func (codec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (codec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (codec) Name() string {
	return Name
}
