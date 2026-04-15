package plc

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/supine-win/go-iot-client/core"
)

type AllenBradleyClient struct {
	mu       sync.Mutex
	endpoint string
	slot     byte
	timeout  time.Duration
	retry    core.RetryPolicy
	conn     net.Conn
	session  uint32
}

func NewAllenBradleyClient(endpoint string) *AllenBradleyClient {
	return &AllenBradleyClient{
		endpoint: endpoint,
		slot:     0,
		timeout:  1500 * time.Millisecond,
		retry: core.RetryPolicy{
			MaxRetries: 2,
			RetryDelay: 100 * time.Millisecond,
		}.Normalize(),
	}
}

func (c *AllenBradleyClient) SetSlot(slot byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.slot = slot
}

func (c *AllenBradleyClient) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if timeout > 0 {
		c.timeout = timeout
	}
}

func (c *AllenBradleyClient) SetRetryPolicy(policy core.RetryPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retry = policy.Normalize()
}

func (c *AllenBradleyClient) Open() core.Result {
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
	if err := c.registerSessionLocked(); err != nil {
		_ = c.conn.Close()
		c.conn = nil
		r.IsSucceed = false
		r.Err = err.Error()
		return core.EndResult(r)
	}
	return core.EndResult(r)
}

func (c *AllenBradleyClient) Close() core.Result {
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
	c.session = 0
	return core.EndResult(r)
}

func (c *AllenBradleyClient) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *AllenBradleyClient) ReadInt16(address string) core.ResultT[int16] {
	out := core.ResultT[int16]{Result: core.NewResult()}
	if err := validateAllenTag(address); err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	data, err := c.readRaw(address, 2)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	if len(data) < 2 {
		out.IsSucceed = false
		out.Err = "not enough data"
		return core.EndResultT(out)
	}
	out.Value = int16(binary.LittleEndian.Uint16(data[0:2]))
	return core.EndResultT(out)
}

func (c *AllenBradleyClient) ReadInt32(address string) core.ResultT[int32] {
	out := core.ResultT[int32]{Result: core.NewResult()}
	if err := validateAllenTag(address); err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	data, err := c.readRaw(address, 4)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	if len(data) < 4 {
		out.IsSucceed = false
		out.Err = "not enough data"
		return core.EndResultT(out)
	}
	out.Value = int32(binary.LittleEndian.Uint32(data[0:4]))
	return core.EndResultT(out)
}

func (c *AllenBradleyClient) ReadFloat(address string) core.ResultT[float32] {
	out := core.ResultT[float32]{Result: core.NewResult()}
	if err := validateAllenTag(address); err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	data, err := c.readRaw(address, 4)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	if len(data) < 4 {
		out.IsSucceed = false
		out.Err = "not enough data"
		return core.EndResultT(out)
	}
	out.Value = math.Float32frombits(binary.LittleEndian.Uint32(data[0:4]))
	return core.EndResultT(out)
}

func (c *AllenBradleyClient) ReadString(address string, readLength int) core.ResultT[string] {
	out := core.ResultT[string]{Result: core.NewResult()}
	if err := validateAllenTag(address); err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	if readLength <= 0 {
		out.IsSucceed = false
		out.Err = "readLength must > 0"
		return core.EndResultT(out)
	}
	data, err := c.readRaw(address, readLength)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	if len(data) > readLength {
		data = data[:readLength]
	}
	out.Value = strings.TrimRight(string(data), "\x00")
	return core.EndResultT(out)
}

func (c *AllenBradleyClient) Write(address string, value interface{}) core.Result {
	r := core.NewResult()
	if err := validateAllenTag(address); err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
		return core.EndResult(r)
	}
	typeCode := uint16(0xC4)
	var data []byte
	var count uint16 = 1
	switch v := value.(type) {
	case bool:
		typeCode = 0xC1
		if v {
			data = []byte{0xFF, 0xFF}
		} else {
			data = []byte{0x00, 0x00}
		}
	case byte:
		typeCode = 0xC2
		data = []byte{v, 0x00}
	case int16:
		typeCode = 0xC3
		data = make([]byte, 2)
		binary.LittleEndian.PutUint16(data, uint16(v))
	case uint16:
		typeCode = 0xC3
		data = make([]byte, 2)
		binary.LittleEndian.PutUint16(data, v)
	case int32:
		typeCode = 0xC4
		data = make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(v))
	case uint32:
		typeCode = 0xC4
		data = make([]byte, 4)
		binary.LittleEndian.PutUint32(data, v)
	case float32:
		typeCode = 0xCA
		data = make([]byte, 4)
		binary.LittleEndian.PutUint32(data, math.Float32bits(v))
	case int64:
		typeCode = 0xC5
		data = make([]byte, 8)
		binary.LittleEndian.PutUint64(data, uint64(v))
	case uint64:
		typeCode = 0xC5
		data = make([]byte, 8)
		binary.LittleEndian.PutUint64(data, v)
	case float64:
		typeCode = 0xCB
		data = make([]byte, 8)
		binary.LittleEndian.PutUint64(data, math.Float64bits(v))
	case string:
		typeCode = 0xC4
		data = []byte(v)
		if len(data) > 65534 {
			r.IsSucceed = false
			r.Err = "string value too long"
			return core.EndResult(r)
		}
		if len(data)%2 != 0 {
			data = append(data, 0x00)
		}
		count = uint16(len(data) / 2)
	default:
		r.IsSucceed = false
		r.Err = fmt.Sprintf("unsupported write type %T", value)
		return core.EndResult(r)
	}

	cmd := c.buildWriteCommand(address, typeCode, data, count)
	if _, err := c.sendCommand(cmd); err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
	}
	return core.EndResult(r)
}

func (c *AllenBradleyClient) readRaw(address string, length int) ([]byte, error) {
	if length <= 0 || length > 65535 {
		return nil, fmt.Errorf("length must be in 1..65535")
	}
	cmd := c.buildReadCommand(address, uint16(length))
	resp, err := c.sendCommand(cmd)
	if err != nil {
		return nil, err
	}
	if len(resp) < 46 {
		return nil, fmt.Errorf("invalid read response")
	}
	count := int(binary.LittleEndian.Uint16(resp[38:40]))
	if count < 6 || len(resp) < 46+count-6 {
		return nil, fmt.Errorf("invalid read count")
	}
	data := make([]byte, count-6)
	copy(data, resp[46:46+count-6])
	return data, nil
}

func (c *AllenBradleyClient) sendCommand(cmd []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	policy := c.retry.Normalize()
	var lastErr error
	for i := 0; i <= policy.MaxRetries; i++ {
		resp, err := c.sendCommandOnceLocked(cmd)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		_ = c.closeLocked()
		if i < policy.MaxRetries {
			time.Sleep(policy.RetryDelay)
		}
	}
	return nil, fmt.Errorf("allen-bradley request failed after %d attempts: %w", policy.MaxRetries+1, lastErr)
}

func (c *AllenBradleyClient) sendCommandOnceLocked(cmd []byte) ([]byte, error) {
	if err := c.ensureConnLocked(); err != nil {
		return nil, err
	}
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(cmd); err != nil {
		return nil, err
	}
	head := make([]byte, 24)
	if _, err := io.ReadFull(c.conn, head); err != nil {
		return nil, err
	}
	contentLen := int(binary.LittleEndian.Uint16(head[2:4]))
	content := make([]byte, contentLen)
	if _, err := io.ReadFull(c.conn, content); err != nil {
		return nil, err
	}
	out := make([]byte, 24+len(content))
	copy(out, head)
	copy(out[24:], content)
	return out, nil
}

func (c *AllenBradleyClient) ensureConnLocked() error {
	if c.conn != nil {
		return nil
	}
	conn, err := net.DialTimeout("tcp", c.endpoint, c.timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return c.registerSessionLocked()
}

func (c *AllenBradleyClient) registerSessionLocked() error {
	cmd := []byte{
		0x65, 0x00,
		0x04, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x00, 0x00,
	}
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	if _, err := c.conn.Write(cmd); err != nil {
		return err
	}
	resp := make([]byte, 28)
	if _, err := io.ReadFull(c.conn, resp); err != nil {
		return err
	}
	c.session = binary.LittleEndian.Uint32(resp[4:8])
	if c.session == 0 {
		return fmt.Errorf("invalid session handle")
	}
	return nil
}

func (c *AllenBradleyClient) closeLocked() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.session = 0
		return err
	}
	return nil
}

func (c *AllenBradleyClient) buildReadCommand(address string, length uint16) []byte {
	addr := []byte(address)
	addrAligned := addr
	if len(addrAligned)%2 != 0 {
		addrAligned = append(append([]byte{}, addrAligned...), 0x00)
	}
	out := make([]byte, 9+26+len(addrAligned)+1+24)
	out[0] = 0x6F
	binary.LittleEndian.PutUint16(out[2:4], uint16(len(out)-24))
	binary.LittleEndian.PutUint32(out[4:8], c.session)
	out[24+4] = 0x01
	out[24+6] = 0x02
	out[24+12] = 0xB2
	binary.LittleEndian.PutUint16(out[24+14:24+16], uint16(len(out)-40))
	out[24+16] = 0x52
	out[24+17] = 0x02
	out[24+18] = 0x20
	out[24+19] = 0x06
	out[24+20] = 0x24
	out[24+21] = 0x01
	out[24+22] = 0x0A
	out[24+23] = 0xF0
	binary.LittleEndian.PutUint16(out[24+24:24+26], uint16(6+len(addrAligned)))

	svc := 24 + 26
	out[svc] = 0x4C
	out[svc+1] = byte((len(addrAligned) + 2) / 2)
	out[svc+2] = 0x91
	out[svc+3] = byte(len(addr))
	copy(out[svc+4:svc+4+len(addrAligned)], addrAligned)
	binary.LittleEndian.PutUint16(out[svc+4+len(addrAligned):svc+6+len(addrAligned)], length)
	out[svc+6+len(addrAligned)] = 0x01
	out[svc+8+len(addrAligned)] = 0x01
	out[svc+9+len(addrAligned)] = c.slot
	return out
}

func (c *AllenBradleyClient) buildWriteCommand(address string, typeCode uint16, data []byte, count uint16) []byte {
	addr := []byte(address)
	addrAligned := addr
	if len(addrAligned)%2 != 0 {
		addrAligned = append(append([]byte{}, addrAligned...), 0x00)
	}
	out := make([]byte, 8+26+len(addrAligned)+len(data)+4+24)
	out[0] = 0x6F
	binary.LittleEndian.PutUint16(out[2:4], uint16(len(out)-24))
	binary.LittleEndian.PutUint32(out[4:8], c.session)
	out[24+4] = 0x01
	out[24+6] = 0x02
	out[24+12] = 0xB2
	binary.LittleEndian.PutUint16(out[24+14:24+16], uint16(len(out)-40))
	out[24+16] = 0x52
	out[24+17] = 0x02
	out[24+18] = 0x20
	out[24+19] = 0x06
	out[24+20] = 0x24
	out[24+21] = 0x01
	out[24+22] = 0x0A
	out[24+23] = 0xF0
	binary.LittleEndian.PutUint16(out[24+24:24+26], uint16(8+len(data)+len(addrAligned)))

	svc := 24 + 26
	out[svc] = 0x4D
	out[svc+1] = byte((len(addrAligned) + 2) / 2)
	out[svc+2] = 0x91
	out[svc+3] = byte(len(addr))
	copy(out[svc+4:svc+4+len(addrAligned)], addrAligned)
	binary.LittleEndian.PutUint16(out[svc+4+len(addrAligned):svc+6+len(addrAligned)], typeCode)
	binary.LittleEndian.PutUint16(out[svc+6+len(addrAligned):svc+8+len(addrAligned)], count)
	copy(out[svc+8+len(addrAligned):svc+8+len(addrAligned)+len(data)], data)
	out[svc+8+len(addrAligned)+len(data)] = 0x01
	out[svc+10+len(addrAligned)+len(data)] = 0x01
	out[svc+11+len(addrAligned)+len(data)] = c.slot
	return out
}

func validateAllenTag(address string) error {
	tag := strings.TrimSpace(address)
	if tag == "" {
		return fmt.Errorf("%w: empty tag", core.ErrInvalidAddress)
	}
	if len(tag) > 255 {
		return fmt.Errorf("%w: tag length exceeds 255", core.ErrInvalidAddress)
	}
	return nil
}
