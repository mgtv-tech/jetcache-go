package msgpack

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testMessage struct {
	Id    int64
	Name  string
	Hobby []string
}

func TestName(t *testing.T) {
	c := new(codec)
	if !reflect.DeepEqual(c.Name(), "msgpack") {
		t.Errorf("no expect float_key value: %v, but got: %v", c.Name(), "msgpack")
	}
}

func TestMsgpackCodec(t *testing.T) {
	tests := []testMessage{
		{
			Id:   1,
			Name: "jetcache-go",
		},
		{
			Id:    1,
			Name:  strings.Repeat("my very large string", 10),
			Hobby: []string{"study", "eat", "play"},
		},
	}

	for _, v := range tests {
		data, err := (codec{}).Marshal(&v)
		assert.Nilf(t, err, "Marshal() should be nil, but got %s", err)

		var res testMessage
		err = (codec{}).Unmarshal(data, &res)
		assert.Nilf(t, err, "Unmarshal() should be nil, but got %s", err)
		if !reflect.DeepEqual(res.Id, v.Id) {
			t.Errorf("ID should be %d, but got %d", res.Id, v.Id)
		}
		if !reflect.DeepEqual(res.Name, v.Name) {
			t.Errorf("Name should be %s, but got %s", res.Name, v.Name)
		}
		if !reflect.DeepEqual(res.Hobby, v.Hobby) {
			t.Errorf("Hobby should be %s, but got %s", res.Hobby, v.Hobby)
		}
	}
}
