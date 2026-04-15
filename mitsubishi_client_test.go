package iotclient

import (
	"fmt"
	"testing"
	"time"

	"github.com/supine-win/go-iot-client/mock"
)

func setupClient(t *testing.T) (*MitsubishiClient, *mock.Server, func()) {
	t.Helper()
	srv, err := mock.NewServer()
	if err != nil {
		t.Fatalf("new mock server: %v", err)
	}
	srv.Start()
	host, port := mock.MustParseAddr(srv.Addr())
	cl := NewMitsubishiClient(MitsubishiVersionQna3E, host, port, 1000)
	if r := cl.Open(); !r.IsSucceed {
		t.Fatalf("open failed: %s", r.Err)
	}
	cleanup := func() {
		_ = cl.Close()
		srv.Close()
	}
	return cl, srv, cleanup
}

func TestReadWriteInt16(t *testing.T) {
	cl, srv, cleanup := setupClient(t)
	defer cleanup()

	srv.SetWord(4001, 100)
	r := cl.ReadInt16("D4001")
	if !r.IsSucceed || r.Value != 100 {
		t.Fatalf("read int16 failed: %+v", r)
	}

	w := cl.Write("D4001", int16(200))
	if !w.IsSucceed {
		t.Fatalf("write int16 failed: %+v", w)
	}
	if got := srv.GetWord(4001); got != 200 {
		t.Fatalf("expect 200 got %d", got)
	}
}

func TestReadFloatAndString(t *testing.T) {
	cl, srv, cleanup := setupClient(t)
	defer cleanup()

	// 12.5 float32 bits
	srv.SetWord(4003, 0x0000)
	srv.SetWord(4004, 0x4148)
	f := cl.ReadFloat("D4003")
	if !f.IsSucceed || fmt.Sprintf("%.2f", f.Value) != "12.50" {
		t.Fatalf("read float failed: %+v", f)
	}

	// "ABCD"
	srv.SetWord(4050, uint16('A')|uint16('B')<<8)
	srv.SetWord(4051, uint16('C')|uint16('D')<<8)
	s := cl.ReadString("D4050", 4)
	if !s.IsSucceed || s.Value != "ABCD" {
		t.Fatalf("read string failed: %+v", s)
	}
}

func TestRetryPolicyReconnect(t *testing.T) {
	cl, srv, cleanup := setupClient(t)
	defer cleanup()

	cl.SetRetryPolicy(2, 10*time.Millisecond)
	srv.SetWord(4001, 123)
	srv.FailNextRequest(0xC123)
	r := cl.ReadInt16("D4001")
	if !r.IsSucceed || r.Value != 123 {
		t.Fatalf("retry reconnect failed: %+v", r)
	}
}

func TestInvalidAddress(t *testing.T) {
	cl, _, cleanup := setupClient(t)
	defer cleanup()

	r := cl.ReadInt16("M100")
	if r.IsSucceed {
		t.Fatalf("expect invalid address error")
	}
}

