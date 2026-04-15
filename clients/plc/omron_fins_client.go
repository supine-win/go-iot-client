package plc

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/example/go-iotclient/core"
)

const defaultOmronTimeout = 1500 * time.Millisecond

type OmronFinsClient struct {
	mu       sync.Mutex
	endpoint string
	timeout  time.Duration
	retry    core.RetryPolicy
	conn     net.Conn

	unitAddress byte
	sa1         byte
	da1         byte
}

func NewOmronFinsClient(endpoint string) *OmronFinsClient {
	return &OmronFinsClient{
		endpoint: endpoint,
		timeout:  defaultOmronTimeout,
		retry: core.RetryPolicy{
			MaxRetries: 2,
			RetryDelay: 100 * time.Millisecond,
		}.Normalize(),
		unitAddress: 0x00,
		sa1:         0x0B,
		da1:         0x01,
	}
}

func (c *OmronFinsClient) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if timeout > 0 {
		c.timeout = timeout
	}
}

func (c *OmronFinsClient) SetRetryPolicy(policy core.RetryPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retry = policy.Normalize()
}

func (c *OmronFinsClient) SetRouting(sa1, da1, unitAddress byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sa1 = sa1
	c.da1 = da1
	c.unitAddress = unitAddress
}

func (c *OmronFinsClient) Open() core.Result {
	r := core.NewResult()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	conn, err := net.DialTimeout("tcp", c.endpoint, c.timeout)
	if err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
		r.ErrCode = 408
		return core.EndResult(r)
	}
	c.conn = conn
	if err := c.handshakeLocked(); err != nil {
		_ = c.conn.Close()
		c.conn = nil
		r.IsSucceed = false
		r.Err = err.Error()
		return core.EndResult(r)
	}
	return core.EndResult(r)
}

func (c *OmronFinsClient) Close() core.Result {
	r := core.NewResult()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			r.IsSucceed = false
			r.Err = err.Error()
		}
		c.conn = nil
	}
	return core.EndResult(r)
}

func (c *OmronFinsClient) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *OmronFinsClient) ReadInt16(address string) core.ResultT[int16] {
	out := core.ResultT[int16]{Result: core.NewResult()}
	words, err := c.readWords(address, 1)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	out.Value = int16(words[0])
	return core.EndResultT(out)
}

func (c *OmronFinsClient) ReadInt32(address string) core.ResultT[int32] {
	out := core.ResultT[int32]{Result: core.NewResult()}
	words, err := c.readWords(address, 2)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	var b [4]byte
	binary.BigEndian.PutUint16(b[0:2], words[0])
	binary.BigEndian.PutUint16(b[2:4], words[1])
	out.Value = int32(binary.BigEndian.Uint32(b[:]))
	return core.EndResultT(out)
}

func (c *OmronFinsClient) ReadFloat(address string) core.ResultT[float32] {
	out := core.ResultT[float32]{Result: core.NewResult()}
	words, err := c.readWords(address, 2)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	var b [4]byte
	binary.BigEndian.PutUint16(b[0:2], words[0])
	binary.BigEndian.PutUint16(b[2:4], words[1])
	out.Value = math.Float32frombits(binary.BigEndian.Uint32(b[:]))
	return core.EndResultT(out)
}

func (c *OmronFinsClient) ReadString(address string, readLength int) core.ResultT[string] {
	out := core.ResultT[string]{Result: core.NewResult()}
	if readLength <= 0 {
		out.IsSucceed = false
		out.Err = "readLength must > 0"
		return core.EndResultT(out)
	}
	wordCount := uint16((readLength + 1) / 2)
	words, err := c.readWords(address, wordCount)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	buf := make([]byte, len(words)*2)
	for i, w := range words {
		binary.BigEndian.PutUint16(buf[i*2:i*2+2], w)
	}
	if len(buf) > readLength {
		buf = buf[:readLength]
	}
	out.Value = strings.TrimRight(string(buf), "\x00")
	return core.EndResultT(out)
}

func (c *OmronFinsClient) Write(address string, value interface{}) core.Result {
	r := core.NewResult()
	var words []uint16
	switch v := value.(type) {
	case bool:
		if v {
			words = []uint16{1}
		} else {
			words = []uint16{0}
		}
	case int16:
		words = []uint16{uint16(v)}
	case uint16:
		words = []uint16{v}
	case int32:
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(v))
		words = []uint16{
			binary.BigEndian.Uint16(b[0:2]),
			binary.BigEndian.Uint16(b[2:4]),
		}
	case uint32:
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], v)
		words = []uint16{
			binary.BigEndian.Uint16(b[0:2]),
			binary.BigEndian.Uint16(b[2:4]),
		}
	case float32:
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], math.Float32bits(v))
		words = []uint16{
			binary.BigEndian.Uint16(b[0:2]),
			binary.BigEndian.Uint16(b[2:4]),
		}
	case string:
		raw := []byte(v)
		if len(raw)%2 != 0 {
			raw = append(raw, 0x00)
		}
		words = make([]uint16, len(raw)/2)
		for i := 0; i < len(words); i++ {
			words[i] = binary.BigEndian.Uint16(raw[i*2 : i*2+2])
		}
	default:
		r.IsSucceed = false
		r.Err = fmt.Sprintf("unsupported write type %T", value)
		return core.EndResult(r)
	}
	if err := c.writeWords(address, words); err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
	}
	return core.EndResult(r)
}

type omronAddr struct {
	wordCode byte
	wordAddr uint16
	bitAddr  byte
}

func parseOmronAddress(address string) (omronAddr, error) {
	addr := strings.ToUpper(strings.TrimSpace(address))
	if addr == "" {
		return omronAddr{}, core.ErrInvalidAddress
	}
	var code byte
	switch addr[0] {
	case 'D':
		code = 0x82
	case 'C':
		code = 0xB0
	case 'W':
		code = 0xB1
	case 'H':
		code = 0xB2
	case 'A':
		code = 0xB3
	default:
		return omronAddr{}, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
	}
	var raw string
	if len(addr) > 1 {
		raw = addr[1:]
	}
	parts := strings.Split(raw, ".")
	word, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return omronAddr{}, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
	}
	out := omronAddr{wordCode: code, wordAddr: uint16(word), bitAddr: 0}
	if len(parts) > 1 {
		bit, bitErr := strconv.ParseUint(parts[1], 10, 8)
		if bitErr != nil || bit > 15 {
			return omronAddr{}, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
		}
		out.bitAddr = byte(bit)
	}
	return out, nil
}

func (c *OmronFinsClient) readWords(address string, count uint16) ([]uint16, error) {
	arg, err := parseOmronAddress(address)
	if err != nil {
		return nil, err
	}
	cmd := c.buildMemoryCommand(0x01, 0x01, arg.wordCode, arg.wordAddr, arg.bitAddr, count, nil)
	resp, err := c.sendCommand(cmd)
	if err != nil {
		return nil, err
	}
	if len(resp) < int(count)*2 {
		return nil, fmt.Errorf("fins response too short")
	}
	words := make([]uint16, count)
	for i := 0; i < int(count); i++ {
		words[i] = binary.BigEndian.Uint16(resp[i*2 : i*2+2])
	}
	return words, nil
}

func (c *OmronFinsClient) writeWords(address string, words []uint16) error {
	arg, err := parseOmronAddress(address)
	if err != nil {
		return err
	}
	data := make([]byte, len(words)*2)
	for i, w := range words {
		binary.BigEndian.PutUint16(data[i*2:i*2+2], w)
	}
	cmd := c.buildMemoryCommand(0x01, 0x02, arg.wordCode, arg.wordAddr, arg.bitAddr, uint16(len(words)), data)
	_, err = c.sendCommand(cmd)
	return err
}

func (c *OmronFinsClient) buildMemoryCommand(mainCmd, subCmd, memCode byte, wordAddr uint16, bitAddr byte, count uint16, data []byte) []byte {
	total := 34 + len(data)
	buf := make([]byte, total)
	copy(buf[0:4], []byte{'F', 'I', 'N', 'S'})
	binary.BigEndian.PutUint32(buf[4:8], uint32(total-8))
	buf[11] = 0x02
	buf[16] = 0x80
	buf[18] = 0x02
	buf[20] = c.da1
	buf[21] = c.unitAddress
	buf[23] = c.sa1
	buf[26] = mainCmd
	buf[27] = subCmd
	buf[28] = memCode
	binary.BigEndian.PutUint16(buf[29:31], wordAddr)
	buf[31] = bitAddr
	binary.BigEndian.PutUint16(buf[32:34], count)
	copy(buf[34:], data)
	return buf
}

func (c *OmronFinsClient) sendCommand(cmd []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	policy := c.retry.Normalize()
	var lastErr error
	for i := 0; i <= policy.MaxRetries; i++ {
		body, err := c.sendCommandOnceLocked(cmd)
		if err == nil {
			return body, nil
		}
		lastErr = err
		_ = c.closeLocked()
		if i < policy.MaxRetries {
			time.Sleep(policy.RetryDelay)
		}
	}
	return nil, fmt.Errorf("fins request failed after %d attempts: %w", policy.MaxRetries+1, lastErr)
}

func (c *OmronFinsClient) sendCommandOnceLocked(cmd []byte) ([]byte, error) {
	if err := c.ensureConnLocked(); err != nil {
		return nil, err
	}
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(cmd); err != nil {
		return nil, err
	}
	head := make([]byte, 8)
	if _, err := io.ReadFull(c.conn, head); err != nil {
		return nil, err
	}
	if string(head[0:4]) != "FINS" {
		return nil, fmt.Errorf("invalid fins magic")
	}
	contentLen := binary.BigEndian.Uint32(head[4:8])
	if contentLen < 26 {
		return nil, fmt.Errorf("invalid fins content length")
	}
	content := make([]byte, contentLen)
	if _, err := io.ReadFull(c.conn, content); err != nil {
		return nil, err
	}
	if len(content) < 26 {
		return nil, fmt.Errorf("fins content too short")
	}
	status := binary.BigEndian.Uint16(content[12:14])
	if status != 0 {
		return nil, fmt.Errorf("fins status: 0x%04X", status)
	}
	if len(content) < 26 {
		return nil, fmt.Errorf("fins body too short")
	}
	endCode := binary.BigEndian.Uint16(content[24:26])
	if endCode != 0 {
		return nil, fmt.Errorf("fins end code: 0x%04X", endCode)
	}
	return content[26:], nil
}

func (c *OmronFinsClient) ensureConnLocked() error {
	if c.conn != nil {
		return nil
	}
	conn, err := net.DialTimeout("tcp", c.endpoint, c.timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return c.handshakeLocked()
}

func (c *OmronFinsClient) handshakeLocked() error {
	cmd := []byte{
		0x46, 0x49, 0x4E, 0x53,
		0x00, 0x00, 0x00, 0x0C,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, c.sa1,
	}
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	if _, err := c.conn.Write(cmd); err != nil {
		return err
	}
	head := make([]byte, 8)
	if _, err := io.ReadFull(c.conn, head); err != nil {
		return err
	}
	if string(head[0:4]) != "FINS" {
		return fmt.Errorf("invalid fins magic")
	}
	contentLen := binary.BigEndian.Uint32(head[4:8])
	content := make([]byte, contentLen)
	if _, err := io.ReadFull(c.conn, content); err != nil {
		return err
	}
	if len(content) >= 16 {
		c.da1 = content[15]
	}
	return nil
}

func (c *OmronFinsClient) closeLocked() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
