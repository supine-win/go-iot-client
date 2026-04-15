package plc

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/example/go-iotclient/core"
)

func TestAllenBradleyClientReadWrite(t *testing.T) {
	srv := newAllenMockServer(t)
	defer srv.Close()

	client := NewAllenBradleyClient(srv.Endpoint())
	if r := client.Open(); !r.IsSucceed {
		t.Fatalf("open failed: %s", r.Err)
	}
	defer client.Close()

	if r := client.Write("TagA", int16(456)); !r.IsSucceed {
		t.Fatalf("write int16 failed: %s", r.Err)
	}
	if r := client.Write("TagB", float32(3.14)); !r.IsSucceed {
		t.Fatalf("write float failed: %s", r.Err)
	}
	if r := client.Write("TagC", "OK"); !r.IsSucceed {
		t.Fatalf("write string failed: %s", r.Err)
	}

	if got := client.ReadInt16("TagA"); !got.IsSucceed || got.Value != 456 {
		t.Fatalf("read int16 unexpected: %#v", got)
	}
	if got := client.ReadFloat("TagB"); !got.IsSucceed || got.Value != float32(3.14) {
		t.Fatalf("read float unexpected: %#v", got)
	}
	if got := client.ReadString("TagC", 2); !got.IsSucceed || got.Value != "OK" {
		t.Fatalf("read string unexpected: %#v", got)
	}
}

func TestAllenBradleyClientRetry(t *testing.T) {
	srv := newAllenMockServer(t)
	defer srv.Close()
	srv.SetDropNext()

	client := NewAllenBradleyClient(srv.Endpoint())
	client.SetRetryPolicy(core.RetryPolicy{MaxRetries: 1, RetryDelay: 10 * time.Millisecond})
	got := client.ReadInt16("TagA")
	if !got.IsSucceed {
		t.Fatalf("retry expected succeed, got %s", got.Err)
	}
}

func TestAllenBradleyUnsupportedWriteType(t *testing.T) {
	client := NewAllenBradleyClient("127.0.0.1:65535")
	r := client.Write("TagA", struct{}{})
	if r.IsSucceed {
		t.Fatalf("expected unsupported type failure")
	}
}

func TestAllenBradleyMalformedResponse(t *testing.T) {
	srv := newAllenMockServer(t)
	defer srv.Close()
	srv.SetMalformedNext()

	client := NewAllenBradleyClient(srv.Endpoint())
	client.SetRetryPolicy(core.RetryPolicy{MaxRetries: 0})
	got := client.ReadInt16("TagA")
	if got.IsSucceed {
		t.Fatalf("expected malformed response failure")
	}
}

func TestAllenBradleyInvalidTag(t *testing.T) {
	client := NewAllenBradleyClient("127.0.0.1:65535")
	if got := client.ReadInt16("   "); got.IsSucceed {
		t.Fatalf("expected invalid tag failure")
	}
	longTag := make([]byte, 256)
	for i := range longTag {
		longTag[i] = 'A'
	}
	if got := client.ReadInt16(string(longTag)); got.IsSucceed {
		t.Fatalf("expected long tag failure")
	}
}

type allenMockServer struct {
	t         *testing.T
	ln        net.Listener
	mu        sync.Mutex
	tags      map[string][]byte
	dropNext  bool
	malformed bool
}

func newAllenMockServer(t *testing.T) *allenMockServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	s := &allenMockServer{
		t:    t,
		ln:   ln,
		tags: map[string][]byte{},
	}
	go s.serve()
	return s
}

func (s *allenMockServer) Endpoint() string { return s.ln.Addr().String() }
func (s *allenMockServer) Close()           { _ = s.ln.Close() }
func (s *allenMockServer) SetDropNext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropNext = true
}
func (s *allenMockServer) SetMalformedNext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.malformed = true
}

func (s *allenMockServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *allenMockServer) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		head := make([]byte, 24)
		if _, err := io.ReadFull(conn, head); err != nil {
			return
		}
		contentLen := int(binary.LittleEndian.Uint16(head[2:4]))
		body := make([]byte, contentLen)
		if _, err := io.ReadFull(conn, body); err != nil {
			return
		}

		cmd := binary.LittleEndian.Uint16(head[0:2])
		if cmd == 0x0065 {
			resp := make([]byte, 28)
			binary.LittleEndian.PutUint16(resp[0:2], 0x0065)
			binary.LittleEndian.PutUint16(resp[2:4], 4)
			binary.LittleEndian.PutUint32(resp[4:8], 0x01020304)
			resp[24] = 0x01
			resp[25] = 0x00
			resp[26] = 0x00
			resp[27] = 0x00
			if _, err := conn.Write(resp); err != nil {
				return
			}
			continue
		}

		if s.shouldDrop(conn) {
			return
		}
		if s.shouldMalformed() {
			resp := make([]byte, 28)
			binary.LittleEndian.PutUint16(resp[0:2], 0x006F)
			binary.LittleEndian.PutUint16(resp[2:4], 4)
			binary.LittleEndian.PutUint32(resp[4:8], binary.LittleEndian.Uint32(head[4:8]))
			_, _ = conn.Write(resp)
			continue
		}

		full := make([]byte, 24+len(body))
		copy(full, head)
		copy(full[24:], body)
		service := full[50]
		pathWords := int(full[51])
		tagLen := int(full[53])
		tag := string(full[54 : 54+tagLen])
		fieldStart := 50 + 2 + pathWords*2

		if service == 0x4D {
			dataStart := fieldStart + 4
			dataEnd := len(full) - 4
			if dataEnd < dataStart {
				dataEnd = dataStart
			}
			raw := make([]byte, dataEnd-dataStart)
			copy(raw, full[dataStart:dataEnd])
			s.mu.Lock()
			s.tags[tag] = raw
			s.mu.Unlock()
			resp := buildAllenReply(binary.LittleEndian.Uint32(head[4:8]), nil)
			if _, err := conn.Write(resp); err != nil {
				return
			}
			continue
		}

		readLen := int(binary.LittleEndian.Uint16(full[fieldStart : fieldStart+2]))
		s.mu.Lock()
		raw := append([]byte{}, s.tags[tag]...)
		s.mu.Unlock()
		if len(raw) < readLen {
			padding := make([]byte, readLen-len(raw))
			raw = append(raw, padding...)
		}
		raw = raw[:readLen]
		resp := buildAllenReply(binary.LittleEndian.Uint32(head[4:8]), raw)
		if _, err := conn.Write(resp); err != nil {
			return
		}
	}
}

func (s *allenMockServer) shouldDrop(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropNext {
		s.dropNext = false
		_ = conn.Close()
		return true
	}
	return false
}

func (s *allenMockServer) shouldMalformed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.malformed {
		s.malformed = false
		return true
	}
	return false
}

func buildAllenReply(session uint32, payload []byte) []byte {
	content := make([]byte, 22+len(payload))
	binary.LittleEndian.PutUint16(content[14:16], uint16(6+len(payload)))
	copy(content[22:], payload)

	out := make([]byte, 24+len(content))
	binary.LittleEndian.PutUint16(out[0:2], 0x006F)
	binary.LittleEndian.PutUint16(out[2:4], uint16(len(content)))
	binary.LittleEndian.PutUint32(out[4:8], session)
	copy(out[24:], content)
	return out
}
