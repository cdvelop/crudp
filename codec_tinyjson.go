package crudp

import (
	"github.com/cdvelop/tinyjson"
	. "github.com/cdvelop/tinystring"
)

// tinyjsonCodec adapts TinyJSON to the Codec interface
type tinyjsonCodec struct {
	tj *tinyjson.TinyJSON
}

// getDefaultCodec returns the default codec (tinyjson)
func getDefaultCodec() Codec {
	return &tinyjsonCodec{
		tj: tinyjson.New(),
	}
}

func (c *tinyjsonCodec) Encode(data any) ([]byte, error) {
	return c.tj.Encode(data)
}

func (c *tinyjsonCodec) Decode(data []byte, v any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = Errf("panic in decode: %v", r)
		}
	}()
	return c.tj.Decode(data, v)
}
