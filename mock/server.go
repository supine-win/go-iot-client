package mock

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

const (
	cmdDeviceRead  uint16 = 0x0401
	cmdDeviceWrite uint16 = 0x1401
	subcmdWord     uint16 = 0x0000
	deviceCodeD    byte   = 0xA8
)

type Server struct {
	listener net.Listener
	addr     string

	mu    sync.Mutex
	words map[uint16]uint16
}

func NewServer() (*Server, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	return &Server{
		listener: l,
		addr:     l.Addr().String(),
		words:    make(map[uint16]uint16),
	}, nil
}

func (s *Server) Addr() string { return s.addr }

func (s *Server) Start() {
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				return
			}
			go s.handleConn(conn)
		}
	}()
}

func (s *Server) Close() { _ = s.listener.Close() }

func (s *Server) SetWord(addr uint16, value uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.words[addr] = value
}

func (s *Server) GetWord(addr uint16) uint16 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.words[addr]
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		header := make([]byte, 9)
		if _, err := fullRead(conn, header); err != nil {
			return
		}
		dataLen := binary.LittleEndian.Uint16(header[7:9])
		cmdData := make([]byte, dataLen)
		if _, err := fullRead(conn, cmdData); err != nil {
			return
		}
		resp := s.processRequest(header, cmdData)
		_, _ = conn.Write(resp)
	}
}

func (s *Server) processRequest(header, cmdData []byte) []byte {
	networkNo := header[2]
	stationNo := header[3]
	moduleIO := binary.LittleEndian.Uint16(header[4:6])
	multidrop := header[6]

	resp := make([]byte, 0, 64)
	resp = appendLE16(resp, 0xD000)
	resp = append(resp, networkNo, stationNo)
	resp = appendLE16(resp, moduleIO)
	resp = append(resp, multidrop)

	if len(cmdData) < 6 {
		resp = appendLE16(resp, 2)
		resp = appendLE16(resp, 0xC0E1)
		return resp
	}
	cmd := binary.LittleEndian.Uint16(cmdData[2:4])
	subcmd := binary.LittleEndian.Uint16(cmdData[4:6])
	if subcmd != subcmdWord {
		resp = appendLE16(resp, 2)
		resp = appendLE16(resp, 0xC002)
		return resp
	}

	switch cmd {
	case cmdDeviceRead:
		return s.handleRead(resp, cmdData[6:])
	case cmdDeviceWrite:
		return s.handleWrite(resp, cmdData[6:])
	default:
		resp = appendLE16(resp, 2)
		resp = appendLE16(resp, 0xC001)
		return resp
	}
}

func (s *Server) handleRead(resp []byte, data []byte) []byte {
	if len(data) < 6 {
		resp = appendLE16(resp, 2)
		resp = appendLE16(resp, 0xC0E1)
		return resp
	}
	addr := binary.LittleEndian.Uint16(data[0:2])
	code := data[3]
	count := binary.LittleEndian.Uint16(data[4:6])
	if code != deviceCodeD {
		resp = appendLE16(resp, 2)
		resp = appendLE16(resp, 0xC051)
		return resp
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	body := make([]byte, count*2)
	for i := uint16(0); i < count; i++ {
		v := s.words[addr+i]
		binary.LittleEndian.PutUint16(body[i*2:i*2+2], v)
	}
	resp = appendLE16(resp, uint16(2+len(body)))
	resp = appendLE16(resp, 0x0000)
	resp = append(resp, body...)
	return resp
}

func (s *Server) handleWrite(resp []byte, data []byte) []byte {
	if len(data) < 6 {
		resp = appendLE16(resp, 2)
		resp = appendLE16(resp, 0xC0E1)
		return resp
	}
	addr := binary.LittleEndian.Uint16(data[0:2])
	code := data[3]
	count := binary.LittleEndian.Uint16(data[4:6])
	if code != deviceCodeD {
		resp = appendLE16(resp, 2)
		resp = appendLE16(resp, 0xC051)
		return resp
	}
	if len(data) < 6+int(count)*2 {
		resp = appendLE16(resp, 2)
		resp = appendLE16(resp, 0xC0E1)
		return resp
	}

	s.mu.Lock()
	for i := uint16(0); i < count; i++ {
		v := binary.LittleEndian.Uint16(data[6+i*2 : 6+i*2+2])
		s.words[addr+i] = v
	}
	s.mu.Unlock()

	resp = appendLE16(resp, 2)
	resp = appendLE16(resp, 0x0000)
	return resp
}

func appendLE16(buf []byte, v uint16) []byte {
	return append(buf, byte(v&0xFF), byte(v>>8))
}

func fullRead(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func MustParseAddr(addr string) (string, int) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		panic(fmt.Sprintf("mock: bad address %q: %v", addr, err))
	}
	var p int
	fmt.Sscanf(port, "%d", &p)
	return host, p
}

