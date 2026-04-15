package core

import "errors"

var (
	ErrNotConnected   = errors.New("iotclient: not connected")
	ErrUnsupported    = errors.New("iotclient: unsupported operation")
	ErrInvalidAddress = errors.New("iotclient: invalid address")
)

