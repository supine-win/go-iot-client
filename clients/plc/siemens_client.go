package plc

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/supine-win/go-iot-client/core"
	"github.com/robinson/gos7"
)

type SiemensClient struct {
	mu        sync.Mutex
	endpoint  string
	rack      int
	slot      int
	timeout   time.Duration
	retry     core.RetryPolicy
	handler   *gos7.TCPClientHandler
	client    gos7.Client
	connected bool
}

func NewSiemensClient(endpoint string) *SiemensClient {
	return &SiemensClient{
		endpoint: endpoint,
		rack:     0,
		slot:     0,
		timeout:  1500 * time.Millisecond,
		retry: core.RetryPolicy{
			MaxRetries: 2,
			RetryDelay: 100 * time.Millisecond,
		}.Normalize(),
	}
}

func (c *SiemensClient) SetRackSlot(rack, slot int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if rack >= 0 {
		c.rack = rack
	}
	if slot >= 0 {
		c.slot = slot
	}
}

func (c *SiemensClient) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if timeout > 0 {
		c.timeout = timeout
	}
}

func (c *SiemensClient) SetRetryPolicy(policy core.RetryPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retry = policy.Normalize()
}

func (c *SiemensClient) Open() core.Result {
	r := core.NewResult()
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.openLocked(); err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
		r.ErrCode = 408
	}
	return core.EndResult(r)
}

func (c *SiemensClient) Close() core.Result {
	r := core.NewResult()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.handler != nil {
		if err := c.handler.Close(); err != nil {
			r.IsSucceed = false
			r.Err = err.Error()
		}
	}
	c.handler = nil
	c.client = nil
	c.connected = false
	return core.EndResult(r)
}

func (c *SiemensClient) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *SiemensClient) ReadInt16(address string) core.ResultT[int16] {
	out := core.ResultT[int16]{Result: core.NewResult()}
	data, err := c.readBytes(address, 2)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	out.Value = int16(binary.BigEndian.Uint16(data))
	return core.EndResultT(out)
}

func (c *SiemensClient) ReadInt32(address string) core.ResultT[int32] {
	out := core.ResultT[int32]{Result: core.NewResult()}
	data, err := c.readBytes(address, 4)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	out.Value = int32(binary.BigEndian.Uint32(data))
	return core.EndResultT(out)
}

func (c *SiemensClient) ReadFloat(address string) core.ResultT[float32] {
	out := core.ResultT[float32]{Result: core.NewResult()}
	data, err := c.readBytes(address, 4)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	out.Value = math.Float32frombits(binary.BigEndian.Uint32(data))
	return core.EndResultT(out)
}

func (c *SiemensClient) ReadString(address string, readLength int) core.ResultT[string] {
	out := core.ResultT[string]{Result: core.NewResult()}
	if readLength <= 0 || readLength > 65535 {
		out.IsSucceed = false
		out.Err = "readLength must be in 1..65535"
		return core.EndResultT(out)
	}
	data, err := c.readBytes(address, readLength)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	out.Value = strings.TrimRight(string(data), "\x00")
	return core.EndResultT(out)
}

func (c *SiemensClient) Write(address string, value interface{}) core.Result {
	r := core.NewResult()
	var data []byte
	switch v := value.(type) {
	case bool:
		if v {
			data = []byte{0x01}
		} else {
			data = []byte{0x00}
		}
	case byte:
		data = []byte{v}
	case int16:
		data = make([]byte, 2)
		binary.BigEndian.PutUint16(data, uint16(v))
	case uint16:
		data = make([]byte, 2)
		binary.BigEndian.PutUint16(data, v)
	case int32:
		data = make([]byte, 4)
		binary.BigEndian.PutUint32(data, uint32(v))
	case uint32:
		data = make([]byte, 4)
		binary.BigEndian.PutUint32(data, v)
	case int64:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, uint64(v))
	case uint64:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, v)
	case float32:
		data = make([]byte, 4)
		binary.BigEndian.PutUint32(data, math.Float32bits(v))
	case float64:
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, math.Float64bits(v))
	case string:
		data = []byte(v)
	default:
		r.IsSucceed = false
		r.Err = fmt.Sprintf("unsupported write type %T", value)
		return core.EndResult(r)
	}
	if err := c.writeBytes(address, data); err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
	}
	return core.EndResult(r)
}

type siemensAreaType int

const (
	siemensAreaDB siemensAreaType = iota
	siemensAreaM
	siemensAreaI
	siemensAreaQ
)

type siemensAddr struct {
	area siemensAreaType
	db   int
	pos  int
}

func parseSiemensAddress(address string) (siemensAddr, error) {
	addr := strings.ToUpper(strings.TrimSpace(address))
	if addr == "" {
		return siemensAddr{}, core.ErrInvalidAddress
	}
	if strings.HasPrefix(addr, "DB") {
		parts := strings.Split(addr, ".")
		if len(parts) != 2 {
			return siemensAddr{}, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
		}
		dbNo, err := strconv.Atoi(parts[0][2:])
		if err != nil || dbNo < 1 || dbNo > 65535 {
			return siemensAddr{}, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
		}
		part := parts[1]
		var offsetRaw string
		switch {
		case strings.HasPrefix(part, "DBB"):
			offsetRaw = part[3:]
		case strings.HasPrefix(part, "DBW"):
			offsetRaw = part[3:]
		case strings.HasPrefix(part, "DBD"):
			offsetRaw = part[3:]
		default:
			return siemensAddr{}, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
		}
		offset, offErr := strconv.Atoi(offsetRaw)
		if offErr != nil || offset < 0 || offset > 65535 {
			return siemensAddr{}, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
		}
		return siemensAddr{area: siemensAreaDB, db: dbNo, pos: offset}, nil
	}
	prefix := addr[0]
	offset, err := strconv.Atoi(addr[1:])
	if err != nil || offset < 0 || offset > 65535 {
		return siemensAddr{}, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
	}
	switch prefix {
	case 'M':
		return siemensAddr{area: siemensAreaM, pos: offset}, nil
	case 'I':
		return siemensAddr{area: siemensAreaI, pos: offset}, nil
	case 'Q':
		return siemensAddr{area: siemensAreaQ, pos: offset}, nil
	default:
		return siemensAddr{}, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
	}
}

func (c *SiemensClient) readBytes(address string, size int) ([]byte, error) {
	addr, err := parseSiemensAddress(address)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, size)
	op := func() error {
		switch addr.area {
		case siemensAreaDB:
			return c.client.AGReadDB(addr.db, addr.pos, size, buf)
		case siemensAreaM:
			return c.client.AGReadMB(addr.pos, size, buf)
		case siemensAreaI:
			return c.client.AGReadEB(addr.pos, size, buf)
		case siemensAreaQ:
			return c.client.AGReadAB(addr.pos, size, buf)
		default:
			return core.ErrUnsupported
		}
	}
	if err := c.withRetry(op); err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *SiemensClient) writeBytes(address string, data []byte) error {
	addr, err := parseSiemensAddress(address)
	if err != nil {
		return err
	}
	op := func() error {
		switch addr.area {
		case siemensAreaDB:
			return c.client.AGWriteDB(addr.db, addr.pos, len(data), data)
		case siemensAreaM:
			return c.client.AGWriteMB(addr.pos, len(data), data)
		case siemensAreaI:
			return c.client.AGWriteEB(addr.pos, len(data), data)
		case siemensAreaQ:
			return c.client.AGWriteAB(addr.pos, len(data), data)
		default:
			return core.ErrUnsupported
		}
	}
	return c.withRetry(op)
}

func (c *SiemensClient) withRetry(op func() error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	policy := c.retry.Normalize()
	var lastErr error
	for i := 0; i <= policy.MaxRetries; i++ {
		if err := c.ensureConnectedLocked(); err != nil {
			lastErr = err
		} else {
			err := op()
			if err == nil {
				return nil
			}
			lastErr = err
		}
		c.closeLocked()
		if i < policy.MaxRetries {
			time.Sleep(policy.RetryDelay)
		}
	}
	return fmt.Errorf("siemens request failed after %d attempts: %w", policy.MaxRetries+1, lastErr)
}

func (c *SiemensClient) ensureConnectedLocked() error {
	if c.connected && c.client != nil && c.handler != nil {
		return nil
	}
	return c.openLocked()
}

func (c *SiemensClient) openLocked() error {
	c.closeLocked()
	h := gos7.NewTCPClientHandler(c.endpoint, c.rack, c.slot)
	h.Timeout = c.timeout
	h.IdleTimeout = c.timeout * 2
	if err := h.Connect(); err != nil {
		return err
	}
	c.handler = h
	c.client = gos7.NewClient(h)
	c.connected = true
	return nil
}

func (c *SiemensClient) closeLocked() {
	if c.handler != nil {
		_ = c.handler.Close()
	}
	c.handler = nil
	c.client = nil
	c.connected = false
}
