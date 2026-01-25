package codec

import "errors"

var (
	ErrType    = errors.New("codec: parse type error")
	ErrLength  = errors.New("codec: data length mismatch")
	ErrUnknown = errors.New("codec: unknown error")
)
