package clients

import "github.com/supine-win/go-iot-client/core"

type IoTClient interface {
	Open() core.Result
	Close() core.Result
	Connected() bool
	ReadInt16(address string) core.ResultT[int16]
	ReadInt32(address string) core.ResultT[int32]
	ReadFloat(address string) core.ResultT[float32]
	ReadString(address string, readLength int) core.ResultT[string]
	Write(address string, value interface{}) core.Result
}

