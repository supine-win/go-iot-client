package plc

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/supine-win/go-iot-client/core"
)

func TestOmronFinsClientReadWrite(t *testing.T) {
	srv := newOmronMockServer(t)
	defer srv.Close()

	client := NewOmronFinsClient(srv.Endpoint())
	if r := client.Open(); !r.IsSucceed {
		t.Fatalf("open failed: %s", r.Err)
	}
	defer client.Close()

	if r := client.Write("D100", int16(1234)); !r.IsSucceed {
		t.Fatalf("write int16 failed: %s", r.Err)
	}
	if r := client.Write("D110", float32(6.25)); !r.IsSucceed {
		t.Fatalf("write float failed: %s", r.Err)
	}
	if r := client.Write("D120", "OK"); !r.IsSucceed {
		t.Fatalf("write string failed: %s", r.Err)
	}

	if got := client.ReadInt16("D100"); !got.IsSucceed || got.Value != 1234 {
		t.Fatalf("read int16 unexpected: %#v", got)
	}
	if got := client.ReadFloat("D110"); !got.IsSucceed || got.Value != float32(6.25) {
		t.Fatalf("read float unexpected: %#v", got)
	}
	if got := client.ReadString("D120", 2); !got.IsSucceed || got.Value != "OK" {
		t.Fatalf("read string unexpected: %#v", got)
	}
}

func TestOmronFinsRetry(t *testing.T) {
	srv := newOmronMockServer(t)
	defer srv.Close()
	srv.SetDropNext()

	client := NewOmronFinsClient(srv.Endpoint())
	client.SetRetryPolicy(core.RetryPolicy{MaxRetries: 1, RetryDelay: 10 * time.Millisecond})
	got := client.ReadInt16("D0")
	if !got.IsSucceed {
		t.Fatalf("retry expected succeed, got %s", got.Err)
	}
}

func TestOmronFinsInvalidAddress(t *testing.T) {
	client := NewOmronFinsClient("127.0.0.1:65535")
	got := client.ReadInt16("X100")
	if got.IsSucceed {
		t.Fatalf("expected invalid address failure")
	}
}

func TestOmronFinsEndCodeError(t *testing.T) {
	srv := newOmronMockServer(t)
	defer srv.Close()
	srv.SetEndCodeNext(0x0005)

	client := NewOmronFinsClient(srv.Endpoint())
	client.SetRetryPolicy(core.RetryPolicy{MaxRetries: 0})
	got := client.ReadInt16("D0")
	if got.IsSucceed {
		t.Fatalf("expected end code error failure")
	}
}

type omronMockServer struct {
	t        *testing.T
	ln       net.Listener
	mu       sync.Mutex
	words    map[uint16]uint16
	dropNext bool
	endCode  uint16
}

func newOmronMockServer(t *testing.T) *omronMockServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	s := &omronMockServer{
		t:     t,
		ln:    ln,
		words: map[uint16]uint16{},
	}
	go s.serve()
	return s
}

func (s *omronMockServer) Endpoint() string { return s.ln.Addr().String() }
func (s *omronMockServer) Close()           { _ = s.ln.Close() }
func (s *omronMockServer) SetDropNext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropNext = true
}
func (s *omronMockServer) SetEndCodeNext(endCode uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endCode = endCode
}

func (s *omronMockServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *omronMockServer) handleConn(conn net.Conn) {
	defer conn.Close()
	handshakeDone := false
	for {
		head := make([]byte, 8)
		if _, err := io.ReadFull(conn, head); err != nil {
			return
		}
		if string(head[:4]) != "FINS" {
			return
		}
		contentLen := binary.BigEndian.Uint32(head[4:8])
		content := make([]byte, contentLen)
		if _, err := io.ReadFull(conn, content); err != nil {
			return
		}
		if !handshakeDone {
			handshakeDone = true
			resp := make([]byte, 16)
			resp[15] = 0x01 // server node
			if err := writeFinsFrame(conn, resp); err != nil {
				return
			}
			continue
		}
		if s.shouldDrop(conn) {
			return
		}
		if len(content) < 26 {
			return
		}
		mainCmd := content[18]
		subCmd := content[19]
		addr := binary.BigEndian.Uint16(content[21:23])
		count := binary.BigEndian.Uint16(content[24:26])
		payload := []byte{}

		if mainCmd == 0x01 && subCmd == 0x01 {
			payload = make([]byte, int(count)*2)
			s.mu.Lock()
			for i := 0; i < int(count); i++ {
				binary.BigEndian.PutUint16(payload[i*2:i*2+2], s.words[addr+uint16(i)])
			}
			s.mu.Unlock()
		} else if mainCmd == 0x01 && subCmd == 0x02 {
			data := content[26:]
			s.mu.Lock()
			for i := 0; i < int(count); i++ {
				s.words[addr+uint16(i)] = binary.BigEndian.Uint16(data[i*2 : i*2+2])
			}
			s.mu.Unlock()
		}

		resp := make([]byte, 26+len(payload))
		s.mu.Lock()
		endCode := s.endCode
		s.endCode = 0
		s.mu.Unlock()
		binary.BigEndian.PutUint16(resp[24:26], endCode)
		copy(resp[26:], payload)
		if err := writeFinsFrame(conn, resp); err != nil {
			return
		}
	}
}

func (s *omronMockServer) shouldDrop(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropNext {
		s.dropNext = false
		_ = conn.Close()
		return true
	}
	return false
}

func writeFinsFrame(conn net.Conn, content []byte) error {
	head := make([]byte, 8)
	copy(head[:4], []byte{'F', 'I', 'N', 'S'})
	binary.BigEndian.PutUint32(head[4:8], uint32(len(content)))
	if _, err := conn.Write(head); err != nil {
		return err
	}
	_, err := conn.Write(content)
	return err
}
