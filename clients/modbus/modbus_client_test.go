package modbus

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/example/go-iotclient/core"
)

func TestTcpClientReadWrite(t *testing.T) {
	srv := newModbusMockServer(t, mockModeTCP)
	defer srv.Close()

	client := NewTcpClient(srv.Endpoint())
	if r := client.Open(); !r.IsSucceed {
		t.Fatalf("open failed: %s", r.Err)
	}
	defer client.Close()

	if r := client.Write("10", int16(1234)); !r.IsSucceed {
		t.Fatalf("write int16 failed: %s", r.Err)
	}
	if r := client.Write("20", float32(12.5)); !r.IsSucceed {
		t.Fatalf("write float failed: %s", r.Err)
	}
	if r := client.Write("30", "ABCD"); !r.IsSucceed {
		t.Fatalf("write string failed: %s", r.Err)
	}
	if r := client.Write("40", true); !r.IsSucceed {
		t.Fatalf("write coil failed: %s", r.Err)
	}

	if got := client.ReadInt16("10"); !got.IsSucceed || got.Value != 1234 {
		t.Fatalf("read int16 unexpected: %#v", got)
	}
	if got := client.ReadFloat("20"); !got.IsSucceed || got.Value != float32(12.5) {
		t.Fatalf("read float unexpected: %#v", got)
	}
	if got := client.ReadString("30", 4); !got.IsSucceed || got.Value != "ABCD" {
		t.Fatalf("read string unexpected: %#v", got)
	}
}

func TestRtuOverTcpClientReadWrite(t *testing.T) {
	srv := newModbusMockServer(t, mockModeRTUOverTCP)
	defer srv.Close()

	client := NewRtuOverTcpClient(srv.Endpoint())
	if r := client.Open(); !r.IsSucceed {
		t.Fatalf("open failed: %s", r.Err)
	}
	defer client.Close()

	if r := client.Write("50", int32(987654)); !r.IsSucceed {
		t.Fatalf("write int32 failed: %s", r.Err)
	}
	got := client.ReadInt32("50")
	if !got.IsSucceed || got.Value != 987654 {
		t.Fatalf("read int32 unexpected: %#v", got)
	}
}

func TestTcpClientRetryOnBrokenPipe(t *testing.T) {
	srv := newModbusMockServer(t, mockModeTCP)
	defer srv.Close()
	srv.SetDropNextRequest()

	client := NewTcpClient(srv.Endpoint())
	client.SetRetryPolicy(core.RetryPolicy{MaxRetries: 1, RetryDelay: 10 * time.Millisecond})

	got := client.ReadInt16("10")
	if !got.IsSucceed {
		t.Fatalf("expected retry success, got error: %s", got.Err)
	}
}

func TestRtuClientReadWriteWithInjectedDialer(t *testing.T) {
	srv := newModbusMockServer(t, mockModeRTU)
	defer srv.Close()

	client := NewRtuClient("virtual")
	client.inner.endpoint = srv.Endpoint()
	client.inner.dial = tcpDial
	client.SetTimeout(500 * time.Millisecond)

	if r := client.Open(); !r.IsSucceed {
		t.Fatalf("open failed: %s", r.Err)
	}
	defer client.Close()

	if r := client.Write("70", int16(2233)); !r.IsSucceed {
		t.Fatalf("write failed: %s", r.Err)
	}
	got := client.ReadInt16("70")
	if !got.IsSucceed || got.Value != 2233 {
		t.Fatalf("read int16 unexpected: %#v", got)
	}
}

func TestAsciiClientReadWriteWithInjectedDialer(t *testing.T) {
	srv := newModbusMockServer(t, mockModeASCII)
	defer srv.Close()

	client := NewAsciiClient("virtual")
	client.inner.endpoint = srv.Endpoint()
	client.inner.dial = tcpDial
	client.SetTimeout(500 * time.Millisecond)

	if r := client.Open(); !r.IsSucceed {
		t.Fatalf("open failed: %s", r.Err)
	}
	defer client.Close()

	if r := client.Write("80", "ZX"); !r.IsSucceed {
		t.Fatalf("write failed: %s", r.Err)
	}
	got := client.ReadString("80", 2)
	if !got.IsSucceed || got.Value != "ZX" {
		t.Fatalf("read string unexpected: %#v", got)
	}
}

func TestInvalidAddress(t *testing.T) {
	client := NewTcpClient("127.0.0.1:65535")
	got := client.ReadInt16("D10")
	if got.IsSucceed {
		t.Fatalf("expected invalid address error")
	}
}

func TestTcpClientExceptionResponse(t *testing.T) {
	srv := newModbusMockServer(t, mockModeTCP)
	defer srv.Close()
	srv.SetExceptionNext(0x03, 0x02)

	client := NewTcpClient(srv.Endpoint())
	client.SetRetryPolicy(core.RetryPolicy{MaxRetries: 0, RetryDelay: 10 * time.Millisecond})
	got := client.ReadInt16("10")
	if got.IsSucceed {
		t.Fatalf("expected exception response to fail")
	}
}

func TestRtuCrcMismatch(t *testing.T) {
	srv := newModbusMockServer(t, mockModeRTU)
	defer srv.Close()
	srv.SetCorruptCRCNext()

	client := NewRtuClient("virtual")
	client.inner.endpoint = srv.Endpoint()
	client.inner.dial = tcpDial
	client.SetRetryPolicy(core.RetryPolicy{MaxRetries: 0, RetryDelay: 10 * time.Millisecond})
	got := client.ReadInt16("10")
	if got.IsSucceed {
		t.Fatalf("expected CRC mismatch to fail")
	}
}

func TestAsciiLrcMismatch(t *testing.T) {
	srv := newModbusMockServer(t, mockModeASCII)
	defer srv.Close()
	srv.SetCorruptLRCNext()

	client := NewAsciiClient("virtual")
	client.inner.endpoint = srv.Endpoint()
	client.inner.dial = tcpDial
	client.SetRetryPolicy(core.RetryPolicy{MaxRetries: 0, RetryDelay: 10 * time.Millisecond})
	got := client.ReadInt16("10")
	if got.IsSucceed {
		t.Fatalf("expected LRC mismatch to fail")
	}
}

type mockMode int

const (
	mockModeTCP mockMode = iota
	mockModeRTUOverTCP
	mockModeRTU
	mockModeASCII
)

type modbusMockServer struct {
	t        *testing.T
	ln       net.Listener
	mode     mockMode
	mu       sync.Mutex
	regs     map[uint16]uint16
	coils    map[uint16]bool
	dropNext bool

	exceptionNext bool
	exFunc        byte
	exCode        byte
	corruptCRC    bool
	corruptLRC    bool
}

func newModbusMockServer(t *testing.T, mode mockMode) *modbusMockServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &modbusMockServer{
		t:     t,
		ln:    ln,
		mode:  mode,
		regs:  map[uint16]uint16{},
		coils: map[uint16]bool{},
	}
	go s.serve()
	return s
}

func (s *modbusMockServer) Endpoint() string { return s.ln.Addr().String() }
func (s *modbusMockServer) Close()           { _ = s.ln.Close() }
func (s *modbusMockServer) SetDropNextRequest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropNext = true
}
func (s *modbusMockServer) SetExceptionNext(functionCode, exceptionCode byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exceptionNext = true
	s.exFunc = functionCode
	s.exCode = exceptionCode
}
func (s *modbusMockServer) SetCorruptCRCNext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.corruptCRC = true
}
func (s *modbusMockServer) SetCorruptLRCNext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.corruptLRC = true
}

func (s *modbusMockServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *modbusMockServer) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		switch s.mode {
		case mockModeTCP:
			if err := s.handleTCP(conn); err != nil {
				return
			}
		case mockModeRTUOverTCP, mockModeRTU:
			if err := s.handleRTU(conn); err != nil {
				return
			}
		case mockModeASCII:
			if err := s.handleASCII(conn); err != nil {
				return
			}
		}
	}
}

func (s *modbusMockServer) maybeDrop(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropNext {
		s.dropNext = false
		_ = conn.Close()
		return true
	}
	return false
}

func (s *modbusMockServer) handleTCP(conn net.Conn) error {
	head := make([]byte, 7)
	if _, err := io.ReadFull(conn, head); err != nil {
		return err
	}
	if s.maybeDrop(conn) {
		return io.EOF
	}
	length := int(binary.BigEndian.Uint16(head[4:6]))
	if length < 2 {
		return io.EOF
	}
	body := make([]byte, length-1)
	if _, err := io.ReadFull(conn, body); err != nil {
		return err
	}
	respPDU := s.applyPDU(body)
	resp := make([]byte, 7+len(respPDU))
	copy(resp[:2], head[:2])
	binary.BigEndian.PutUint16(resp[2:4], 0)
	binary.BigEndian.PutUint16(resp[4:6], uint16(len(respPDU)+1))
	resp[6] = head[6]
	copy(resp[7:], respPDU)
	_, err := conn.Write(resp)
	return err
}

func (s *modbusMockServer) handleRTU(conn net.Conn) error {
	first := make([]byte, 2)
	if _, err := io.ReadFull(conn, first); err != nil {
		return err
	}
	if s.maybeDrop(conn) {
		return io.EOF
	}
	function := first[1]
	var frame []byte
	switch function {
	case 0x03, 0x04:
		rest := make([]byte, 4)
		if _, err := io.ReadFull(conn, rest); err != nil {
			return err
		}
		frame = append(append(first, rest...), make([]byte, 2)...)
	case 0x05:
		rest := make([]byte, 4)
		if _, err := io.ReadFull(conn, rest); err != nil {
			return err
		}
		frame = append(append(first, rest...), make([]byte, 2)...)
	case 0x10:
		header := make([]byte, 5)
		if _, err := io.ReadFull(conn, header); err != nil {
			return err
		}
		byteCount := int(header[4])
		dataCRC := make([]byte, byteCount+2)
		if _, err := io.ReadFull(conn, dataCRC); err != nil {
			return err
		}
		frame = append(frame, first...)
		frame = append(frame, header...)
		frame = append(frame, dataCRC...)
	default:
		return io.EOF
	}
	pdu := frame[1 : len(frame)-2]
	respPDU := s.applyPDU(pdu)
	resp := make([]byte, 0, len(respPDU)+3)
	resp = append(resp, frame[0])
	resp = append(resp, respPDU...)
	crc := crc16(resp)
	s.mu.Lock()
	if s.corruptCRC {
		s.corruptCRC = false
		crc ^= 0x0001
	}
	s.mu.Unlock()
	resp = append(resp, byte(crc&0xFF), byte(crc>>8))
	_, err := conn.Write(resp)
	return err
}

func (s *modbusMockServer) handleASCII(conn net.Conn) error {
	line, err := readASCIIResponseLine(conn)
	if err != nil {
		return err
	}
	if s.maybeDrop(conn) {
		return io.EOF
	}
	if len(line) < 3 || line[0] != ':' {
		return io.EOF
	}
	payloadHex := string(line[1 : len(line)-2])
	raw, err := hex.DecodeString(payloadHex)
	if err != nil || len(raw) < 3 {
		return io.EOF
	}
	if lrc(raw[:len(raw)-1]) != raw[len(raw)-1] {
		return io.EOF
	}
	pdu := raw[1 : len(raw)-1]
	respPDU := s.applyPDU(pdu)
	respRaw := make([]byte, 0, len(respPDU)+2)
	respRaw = append(respRaw, raw[0])
	respRaw = append(respRaw, respPDU...)
	sum := lrc(respRaw)
	s.mu.Lock()
	if s.corruptLRC {
		s.corruptLRC = false
		sum ^= 0x01
	}
	s.mu.Unlock()
	respRaw = append(respRaw, sum)
	hexResp := make([]byte, hex.EncodedLen(len(respRaw)))
	hex.Encode(hexResp, respRaw)
	respLine := append([]byte{':'}, hexResp...)
	respLine = append(respLine, '\r', '\n')
	_, err = conn.Write(respLine)
	return err
}

func (s *modbusMockServer) applyPDU(pdu []byte) []byte {
	if len(pdu) == 0 {
		return []byte{0x80, 0x01}
	}
	s.mu.Lock()
	if s.exceptionNext {
		s.exceptionNext = false
		fn, code := s.exFunc, s.exCode
		s.mu.Unlock()
		if fn == 0 {
			fn = pdu[0]
		}
		if code == 0 {
			code = 0x01
		}
		return []byte{fn | 0x80, code}
	}
	s.mu.Unlock()
	switch pdu[0] {
	case 0x03, 0x04:
		if len(pdu) < 5 {
			return []byte{pdu[0] | 0x80, 0x03}
		}
		addr := binary.BigEndian.Uint16(pdu[1:3])
		count := binary.BigEndian.Uint16(pdu[3:5])
		resp := make([]byte, 2+count*2)
		resp[0] = pdu[0]
		resp[1] = byte(count * 2)
		s.mu.Lock()
		for i := 0; i < int(count); i++ {
			binary.BigEndian.PutUint16(resp[2+i*2:4+i*2], s.regs[addr+uint16(i)])
		}
		s.mu.Unlock()
		return resp
	case 0x05:
		addr := binary.BigEndian.Uint16(pdu[1:3])
		val := binary.BigEndian.Uint16(pdu[3:5]) == 0xFF00
		s.mu.Lock()
		s.coils[addr] = val
		s.mu.Unlock()
		return append([]byte{}, pdu[:5]...)
	case 0x10:
		addr := binary.BigEndian.Uint16(pdu[1:3])
		count := binary.BigEndian.Uint16(pdu[3:5])
		data := pdu[6:]
		s.mu.Lock()
		for i := 0; i < int(count); i++ {
			s.regs[addr+uint16(i)] = binary.BigEndian.Uint16(data[i*2 : i*2+2])
		}
		s.mu.Unlock()
		return []byte{0x10, pdu[1], pdu[2], pdu[3], pdu[4]}
	default:
		return []byte{pdu[0] | 0x80, 0x01}
	}
}
