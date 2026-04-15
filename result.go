package iotclient

import "github.com/supine-win/go-iot-client/core"

type Result = core.Result

type ResultT[T any] struct {
	Result
	Value T `json:"value"`
}

func newResult() Result {
	return core.NewResult()
}

func endResult(r Result) Result {
	return core.EndResult(r)
}

func endResultT[T any](r ResultT[T]) ResultT[T] {
	ended := core.EndResultT(core.ResultT[T]{
		Result: r.Result,
		Value:  r.Value,
	})
	return ResultT[T]{
		Result: ended.Result,
		Value:  ended.Value,
	}
}

