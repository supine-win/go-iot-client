package iotclient

import "time"

type Result struct {
	IsSucceed    bool      `json:"isSucceed"`
	Err          string    `json:"err,omitempty"`
	ErrCode      int       `json:"errCode,omitempty"`
	Request      string    `json:"request,omitempty"`
	Response     string    `json:"response,omitempty"`
	InitialTime  time.Time `json:"initialTime"`
	TimeConsuming float64  `json:"timeConsuming"`
}

type ResultT[T any] struct {
	Result
	Value T `json:"value"`
}

func newResult() Result {
	return Result{
		IsSucceed:   true,
		InitialTime: time.Now(),
	}
}

func endResult(r Result) Result {
	r.TimeConsuming = time.Since(r.InitialTime).Seconds() * 1000
	return r
}

func endResultT[T any](r ResultT[T]) ResultT[T] {
	r.TimeConsuming = time.Since(r.InitialTime).Seconds() * 1000
	return r
}

