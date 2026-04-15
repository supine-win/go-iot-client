package iotclient

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MitsubishiClient struct {
	mu      sync.Mutex
	version MitsubishiVersion
	ip      string
	port    int
	timeout time.Duration

	networkNo        byte
	stationNo        byte
	moduleIO         uint16
	multidropStation byte
	monitoringTimer  uint16
	readTimeout      time.Duration
	writeTimeout     time.Duration
	maxRetries       int
	retryDelay       time.Duration

	conn net.Conn
}

func NewMitsubishiClient(version MitsubishiVersion, ip string, port int, timeoutMs int) *MitsubishiClient {
	if timeoutMs <= 0 {
		timeoutMs = 1500
	}
	if version == "" {
		version = MitsubishiVersionQna3E
	}
	return &MitsubishiClient{
		version:          version,
		ip:               ip,
		port:             port,
		timeout:          time.Duration(timeoutMs) * time.Millisecond,
		networkNo:        0x00,
		stationNo:        0xFF,
		moduleIO:         0x03FF,
		multidropStation: 0x00,
		monitoringTimer:  2000,
		readTimeout:      5 * time.Second,
		writeTimeout:     5 * time.Second,
		maxRetries:       2,
		retryDelay:       100 * time.Millisecond,
	}
}

func (c *MitsubishiClient) SetRoute(networkNo, stationNo byte, moduleIO uint16, multidropStation byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.networkNo = networkNo
	c.stationNo = stationNo
	c.moduleIO = moduleIO
	c.multidropStation = multidropStation
}

func (c *MitsubishiClient) SetMonitoringTimer(ms uint16) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.monitoringTimer = ms
}

func (c *MitsubishiClient) SetReadWriteTimeout(readTimeout, writeTimeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if readTimeout > 0 {
		c.readTimeout = readTimeout
	}
	if writeTimeout > 0 {
		c.writeTimeout = writeTimeout
	}
}

func (c *MitsubishiClient) SetRetryPolicy(maxRetries int, retryDelay time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if maxRetries < 0 {
		maxRetries = 0
	}
	c.maxRetries = maxRetries
	if retryDelay > 0 {
		c.retryDelay = retryDelay
	}
}

func (c *MitsubishiClient) Open() Result {
	r := newResult()
	if c.version != MitsubishiVersionQna3E {
		r.IsSucceed = false
		r.Err = fmt.Sprintf("unsupported version: %s", c.version)
		return endResult(r)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	conn, err := c.dialLocked()
	if err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
		r.ErrCode = 408
		return endResult(r)
	}
	c.conn = conn
	return endResult(r)
}

func (c *MitsubishiClient) Close() Result {
	r := newResult()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return endResult(r)
	}
	if err := c.conn.Close(); err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
	}
	c.conn = nil
	return endResult(r)
}

func (c *MitsubishiClient) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *MitsubishiClient) ReadInt16(address string) ResultT[int16] {
	base := newResult()
	out := ResultT[int16]{Result: base}
	addr, err := parseDAddress(address)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return endResultT(out)
	}
	words, err := c.readWords(addr, 1)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return endResultT(out)
	}
	if len(words) == 0 {
		out.IsSucceed = false
		out.Err = "empty response"
		return endResultT(out)
	}
	out.Value = int16(words[0])
	return endResultT(out)
}

func (c *MitsubishiClient) ReadInt32(address string) ResultT[int32] {
	base := newResult()
	out := ResultT[int32]{Result: base}
	addr, err := parseDAddress(address)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return endResultT(out)
	}
	words, err := c.readWords(addr, 2)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return endResultT(out)
	}
	if len(words) < 2 {
		out.IsSucceed = false
		out.Err = "not enough words"
		return endResultT(out)
	}
	v := uint32(words[0]) | uint32(words[1])<<16
	out.Value = int32(v)
	return endResultT(out)
}

func (c *MitsubishiClient) ReadFloat(address string) ResultT[float32] {
	base := newResult()
	out := ResultT[float32]{Result: base}
	addr, err := parseDAddress(address)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return endResultT(out)
	}
	words, err := c.readWords(addr, 2)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return endResultT(out)
	}
	if len(words) < 2 {
		out.IsSucceed = false
		out.Err = "not enough words"
		return endResultT(out)
	}
	bits := uint32(words[0]) | uint32(words[1])<<16
	out.Value = math.Float32frombits(bits)
	return endResultT(out)
}

func (c *MitsubishiClient) ReadString(address string, readLength int) ResultT[string] {
	base := newResult()
	out := ResultT[string]{Result: base}
	if readLength <= 0 {
		out.IsSucceed = false
		out.Err = "readLength must > 0"
		return endResultT(out)
	}

	addr, err := parseDAddress(address)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return endResultT(out)
	}
	wordCount := uint16((readLength + 1) / 2)
	words, err := c.readWords(addr, wordCount)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return endResultT(out)
	}
	raw := wordsToBytes(words)
	if len(raw) > readLength {
		raw = raw[:readLength]
	}
	out.Value = strings.TrimRight(string(raw), "\x00")
	return endResultT(out)
}

func (c *MitsubishiClient) Write(address string, value interface{}) Result {
	r := newResult()
	addr, err := parseDAddress(address)
	if err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
		return endResult(r)
	}

	var writeErr error
	switch v := value.(type) {
	case bool:
		if v {
			writeErr = c.writeWords(addr, []uint16{1})
		} else {
			writeErr = c.writeWords(addr, []uint16{0})
		}
	case byte:
		writeErr = c.writeWords(addr, []uint16{uint16(v)})
	case int16:
		writeErr = c.writeWords(addr, []uint16{uint16(v)})
	case uint16:
		writeErr = c.writeWords(addr, []uint16{v})
	case int32:
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(v))
		writeErr = c.writeWords(addr, []uint16{
			binary.LittleEndian.Uint16(b[0:2]),
			binary.LittleEndian.Uint16(b[2:4]),
		})
	case uint32:
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, v)
		writeErr = c.writeWords(addr, []uint16{
			binary.LittleEndian.Uint16(b[0:2]),
			binary.LittleEndian.Uint16(b[2:4]),
		})
	case float32:
		bits := math.Float32bits(v)
		writeErr = c.writeWords(addr, []uint16{
			uint16(bits & 0xFFFF),
			uint16(bits >> 16),
		})
	case string:
		writeErr = c.writeWords(addr, bytesToWords([]byte(v)))
	default:
		writeErr = fmt.Errorf("unsupported write type %T", value)
	}
	if writeErr != nil {
		r.IsSucceed = false
		r.Err = writeErr.Error()
	}
	return endResult(r)
}

func (c *MitsubishiClient) readWords(addr uint16, count uint16) ([]uint16, error) {
	if count == 0 {
		return []uint16{}, nil
	}
	payload := make([]byte, 6)
	payload[0] = byte(addr & 0xFF)
	payload[1] = byte(addr >> 8)
	payload[2] = 0x00
	payload[3] = 0xA8 // D register
	payload[4] = byte(count & 0xFF)
	payload[5] = byte(count >> 8)

	resp, err := c.sendRequest(0x0401, 0x0000, payload, c.readTimeout)
	if err != nil {
		return nil, err
	}
	if len(resp) < int(count)*2 {
		return nil, fmt.Errorf("response data too short: want %d bytes got %d", int(count)*2, len(resp))
	}
	words := make([]uint16, count)
	for i := 0; i < int(count); i++ {
		words[i] = binary.LittleEndian.Uint16(resp[i*2 : i*2+2])
	}
	return words, nil
}

func (c *MitsubishiClient) writeWords(addr uint16, values []uint16) error {
	count := uint16(len(values))
	payload := make([]byte, 6+int(count)*2)
	payload[0] = byte(addr & 0xFF)
	payload[1] = byte(addr >> 8)
	payload[2] = 0x00
	payload[3] = 0xA8 // D register
	payload[4] = byte(count & 0xFF)
	payload[5] = byte(count >> 8)
	for i, v := range values {
		binary.LittleEndian.PutUint16(payload[6+i*2:6+i*2+2], v)
	}
	_, err := c.sendRequest(0x1401, 0x0000, payload, c.writeTimeout)
	return err
}

func (c *MitsubishiClient) sendRequest(command uint16, subcommand uint16, payload []byte, timeout time.Duration) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, err := c.sendRequestOnceLocked(command, subcommand, payload, timeout)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		_ = c.reconnectLocked()
		if attempt < c.maxRetries {
			time.Sleep(c.retryDelay)
		}
	}
	return nil, fmt.Errorf("send request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

func (c *MitsubishiClient) sendRequestOnceLocked(command uint16, subcommand uint16, payload []byte, timeout time.Duration) ([]byte, error) {
	if c.conn == nil {
		if err := c.reconnectLocked(); err != nil {
			return nil, err
		}
	}

	dataLen := uint16(2 + 2 + 2 + len(payload))
	req := make([]byte, 0, 9+int(dataLen))
	req = append(req,
		0x50, 0x00,
		c.networkNo,
		c.stationNo,
		byte(c.moduleIO&0xFF), byte(c.moduleIO>>8),
		c.multidropStation,
		byte(dataLen&0xFF), byte(dataLen>>8),
		byte(c.monitoringTimer&0xFF), byte(c.monitoringTimer>>8),
		byte(command&0xFF), byte(command>>8),
		byte(subcommand&0xFF), byte(subcommand>>8),
	)
	req = append(req, payload...)

	if err := c.conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(req); err != nil {
		return nil, err
	}

	head := make([]byte, 9)
	if _, err := readFull(c.conn, head); err != nil {
		return nil, err
	}
	responseLen := binary.LittleEndian.Uint16(head[7:9])
	body := make([]byte, responseLen)
	if _, err := readFull(c.conn, body); err != nil {
		return nil, err
	}
	if len(body) < 2 {
		return nil, fmt.Errorf("response too short")
	}
	endCode := binary.LittleEndian.Uint16(body[0:2])
	if endCode != 0 {
		return nil, fmt.Errorf("plc error endcode=0x%04X", endCode)
	}
	return body[2:], nil
}

func (c *MitsubishiClient) reconnectLocked() error {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	conn, err := c.dialLocked()
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *MitsubishiClient) dialLocked() (net.Conn, error) {
	address := net.JoinHostPort(c.ip, strconv.Itoa(c.port))
	conn, err := net.DialTimeout("tcp", address, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s failed: %w", address, err)
	}
	return conn, nil
}

func parseDAddress(address string) (uint16, error) {
	addr := strings.TrimSpace(strings.ToUpper(address))
	if len(addr) < 2 || addr[0] != 'D' {
		return 0, fmt.Errorf("only D address supported, got %q", address)
	}
	n, err := strconv.ParseUint(addr[1:], 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid D address %q: %w", address, err)
	}
	return uint16(n), nil
}

func bytesToWords(data []byte) []uint16 {
	if len(data)%2 != 0 {
		data = append(data, 0x00)
	}
	words := make([]uint16, len(data)/2)
	for i := 0; i < len(words); i++ {
		words[i] = uint16(data[i*2]) | uint16(data[i*2+1])<<8
	}
	return words
}

func wordsToBytes(words []uint16) []byte {
	out := make([]byte, len(words)*2)
	for i, v := range words {
		out[i*2] = byte(v & 0xFF)
		out[i*2+1] = byte(v >> 8)
	}
	return out
}

func readFull(conn net.Conn, buf []byte) (int, error) {
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

