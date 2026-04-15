package modbus

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/example/go-iotclient/core"
	"go.bug.st/serial"
)

const (
	defaultTimeout = 1500 * time.Millisecond
)

type wireMode int

const (
	modeTCP wireMode = iota
	modeRTUOverTCP
	modeRTU
	modeASCII
)

type deadlineSetter interface {
	SetDeadline(time.Time) error
}

type streamConn interface {
	io.ReadWriteCloser
}

type dialFunc func(c *modbusClient) (streamConn, error)

type modbusClient struct {
	mu        sync.Mutex
	endpoint  string
	mode      wireMode
	unitID    byte
	timeout   time.Duration
	retry     core.RetryPolicy
	transID   uint16
	conn      streamConn
	autoClose bool
	dial      dialFunc

	serialMode serial.Mode
}

type TcpClient struct{ inner *modbusClient }
type RtuOverTcpClient struct{ inner *modbusClient }
type RtuClient struct{ inner *modbusClient }
type AsciiClient struct{ inner *modbusClient }

func NewTcpClient(endpoint string) *TcpClient {
	return &TcpClient{
		inner: &modbusClient{
			endpoint: endpoint,
			mode:     modeTCP,
			unitID:   1,
			timeout:  defaultTimeout,
			retry: core.RetryPolicy{
				MaxRetries: 2,
				RetryDelay: 100 * time.Millisecond,
			}.Normalize(),
			dial: tcpDial,
		},
	}
}

func NewRtuOverTcpClient(endpoint string) *RtuOverTcpClient {
	return &RtuOverTcpClient{
		inner: &modbusClient{
			endpoint: endpoint,
			mode:     modeRTUOverTCP,
			unitID:   1,
			timeout:  defaultTimeout,
			retry: core.RetryPolicy{
				MaxRetries: 2,
				RetryDelay: 100 * time.Millisecond,
			}.Normalize(),
			dial: tcpDial,
		},
	}
}

func NewRtuClient(port string) *RtuClient {
	return &RtuClient{
		inner: &modbusClient{
			endpoint: port,
			mode:     modeRTU,
			unitID:   1,
			timeout:  defaultTimeout,
			retry: core.RetryPolicy{
				MaxRetries: 2,
				RetryDelay: 100 * time.Millisecond,
			}.Normalize(),
			dial: serialDial,
			serialMode: serial.Mode{
				BaudRate: 9600,
				DataBits: 8,
				Parity:   serial.NoParity,
				StopBits: serial.OneStopBit,
			},
		},
	}
}

func NewAsciiClient(port string) *AsciiClient {
	return &AsciiClient{
		inner: &modbusClient{
			endpoint: port,
			mode:     modeASCII,
			unitID:   1,
			timeout:  defaultTimeout,
			retry: core.RetryPolicy{
				MaxRetries: 2,
				RetryDelay: 100 * time.Millisecond,
			}.Normalize(),
			dial: serialDial,
			serialMode: serial.Mode{
				BaudRate: 9600,
				DataBits: 7,
				Parity:   serial.EvenParity,
				StopBits: serial.OneStopBit,
			},
		},
	}
}

func (c *TcpClient) SetUnitID(unitID byte)                   { c.inner.SetUnitID(unitID) }
func (c *TcpClient) SetTimeout(timeout time.Duration)        { c.inner.SetTimeout(timeout) }
func (c *TcpClient) SetRetryPolicy(policy core.RetryPolicy)  { c.inner.SetRetryPolicy(policy) }
func (c *RtuOverTcpClient) SetUnitID(unitID byte)            { c.inner.SetUnitID(unitID) }
func (c *RtuOverTcpClient) SetTimeout(timeout time.Duration) { c.inner.SetTimeout(timeout) }
func (c *RtuOverTcpClient) SetRetryPolicy(policy core.RetryPolicy) {
	c.inner.SetRetryPolicy(policy)
}
func (c *RtuClient) SetUnitID(unitID byte)                    { c.inner.SetUnitID(unitID) }
func (c *RtuClient) SetTimeout(timeout time.Duration)         { c.inner.SetTimeout(timeout) }
func (c *RtuClient) SetRetryPolicy(policy core.RetryPolicy)   { c.inner.SetRetryPolicy(policy) }
func (c *AsciiClient) SetUnitID(unitID byte)                  { c.inner.SetUnitID(unitID) }
func (c *AsciiClient) SetTimeout(timeout time.Duration)       { c.inner.SetTimeout(timeout) }
func (c *AsciiClient) SetRetryPolicy(policy core.RetryPolicy) { c.inner.SetRetryPolicy(policy) }

func (c *RtuClient) SetSerialMode(mode serial.Mode) {
	c.inner.SetSerialMode(mode)
}
func (c *AsciiClient) SetSerialMode(mode serial.Mode) {
	c.inner.SetSerialMode(mode)
}

func (c *TcpClient) Open() core.Result                            { return c.inner.Open() }
func (c *TcpClient) Close() core.Result                           { return c.inner.Close() }
func (c *TcpClient) Connected() bool                              { return c.inner.Connected() }
func (c *TcpClient) ReadInt16(address string) core.ResultT[int16] { return c.inner.ReadInt16(address) }
func (c *TcpClient) ReadInt32(address string) core.ResultT[int32] { return c.inner.ReadInt32(address) }
func (c *TcpClient) ReadFloat(address string) core.ResultT[float32] {
	return c.inner.ReadFloat(address)
}
func (c *TcpClient) ReadString(address string, n int) core.ResultT[string] {
	return c.inner.ReadString(address, n)
}
func (c *TcpClient) Write(address string, value interface{}) core.Result {
	return c.inner.Write(address, value)
}

func (c *RtuOverTcpClient) Open() core.Result  { return c.inner.Open() }
func (c *RtuOverTcpClient) Close() core.Result { return c.inner.Close() }
func (c *RtuOverTcpClient) Connected() bool    { return c.inner.Connected() }
func (c *RtuOverTcpClient) ReadInt16(address string) core.ResultT[int16] {
	return c.inner.ReadInt16(address)
}
func (c *RtuOverTcpClient) ReadInt32(address string) core.ResultT[int32] {
	return c.inner.ReadInt32(address)
}
func (c *RtuOverTcpClient) ReadFloat(address string) core.ResultT[float32] {
	return c.inner.ReadFloat(address)
}
func (c *RtuOverTcpClient) ReadString(address string, n int) core.ResultT[string] {
	return c.inner.ReadString(address, n)
}
func (c *RtuOverTcpClient) Write(address string, value interface{}) core.Result {
	return c.inner.Write(address, value)
}

func (c *RtuClient) Open() core.Result  { return c.inner.Open() }
func (c *RtuClient) Close() core.Result { return c.inner.Close() }
func (c *RtuClient) Connected() bool    { return c.inner.Connected() }
func (c *RtuClient) ReadInt16(address string) core.ResultT[int16] {
	return c.inner.ReadInt16(address)
}
func (c *RtuClient) ReadInt32(address string) core.ResultT[int32] {
	return c.inner.ReadInt32(address)
}
func (c *RtuClient) ReadFloat(address string) core.ResultT[float32] {
	return c.inner.ReadFloat(address)
}
func (c *RtuClient) ReadString(address string, n int) core.ResultT[string] {
	return c.inner.ReadString(address, n)
}
func (c *RtuClient) Write(address string, value interface{}) core.Result {
	return c.inner.Write(address, value)
}

func (c *AsciiClient) Open() core.Result  { return c.inner.Open() }
func (c *AsciiClient) Close() core.Result { return c.inner.Close() }
func (c *AsciiClient) Connected() bool    { return c.inner.Connected() }
func (c *AsciiClient) ReadInt16(address string) core.ResultT[int16] {
	return c.inner.ReadInt16(address)
}
func (c *AsciiClient) ReadInt32(address string) core.ResultT[int32] {
	return c.inner.ReadInt32(address)
}
func (c *AsciiClient) ReadFloat(address string) core.ResultT[float32] {
	return c.inner.ReadFloat(address)
}
func (c *AsciiClient) ReadString(address string, n int) core.ResultT[string] {
	return c.inner.ReadString(address, n)
}
func (c *AsciiClient) Write(address string, value interface{}) core.Result {
	return c.inner.Write(address, value)
}

func (c *modbusClient) SetUnitID(unitID byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if unitID == 0 {
		unitID = 1
	}
	c.unitID = unitID
}

func (c *modbusClient) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if timeout > 0 {
		c.timeout = timeout
	}
}

func (c *modbusClient) SetRetryPolicy(policy core.RetryPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retry = policy.Normalize()
}

func (c *modbusClient) SetSerialMode(mode serial.Mode) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.serialMode = mode
}

func (c *modbusClient) Open() core.Result {
	r := core.NewResult()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	dial := c.dial
	if dial == nil {
		dial = tcpDial
	}
	conn, err := dial(c)
	if err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
		r.ErrCode = 408
		return core.EndResult(r)
	}
	c.conn = conn
	return core.EndResult(r)
}

func (c *modbusClient) Close() core.Result {
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

func (c *modbusClient) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *modbusClient) ReadInt16(address string) core.ResultT[int16] {
	out := core.ResultT[int16]{Result: core.NewResult()}
	addr, err := parseAddress(address)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	values, err := c.readRegisters(addr, 1, 0x03)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	out.Value = int16(values[0])
	return core.EndResultT(out)
}

func (c *modbusClient) ReadInt32(address string) core.ResultT[int32] {
	out := core.ResultT[int32]{Result: core.NewResult()}
	addr, err := parseAddress(address)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	values, err := c.readRegisters(addr, 2, 0x03)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	var b [4]byte
	binary.BigEndian.PutUint16(b[0:2], values[0])
	binary.BigEndian.PutUint16(b[2:4], values[1])
	out.Value = int32(binary.BigEndian.Uint32(b[:]))
	return core.EndResultT(out)
}

func (c *modbusClient) ReadFloat(address string) core.ResultT[float32] {
	out := core.ResultT[float32]{Result: core.NewResult()}
	addr, err := parseAddress(address)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	values, err := c.readRegisters(addr, 2, 0x03)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	var b [4]byte
	binary.BigEndian.PutUint16(b[0:2], values[0])
	binary.BigEndian.PutUint16(b[2:4], values[1])
	out.Value = math.Float32frombits(binary.BigEndian.Uint32(b[:]))
	return core.EndResultT(out)
}

func (c *modbusClient) ReadString(address string, readLength int) core.ResultT[string] {
	out := core.ResultT[string]{Result: core.NewResult()}
	if readLength <= 0 {
		out.IsSucceed = false
		out.Err = "readLength must > 0"
		return core.EndResultT(out)
	}
	addr, err := parseAddress(address)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	count := uint16((readLength + 1) / 2)
	values, err := c.readRegisters(addr, count, 0x03)
	if err != nil {
		out.IsSucceed = false
		out.Err = err.Error()
		return core.EndResultT(out)
	}
	buf := registersToBytes(values)
	if len(buf) > readLength {
		buf = buf[:readLength]
	}
	out.Value = strings.TrimRight(string(buf), "\x00")
	return core.EndResultT(out)
}

func (c *modbusClient) Write(address string, value interface{}) core.Result {
	r := core.NewResult()
	addr, err := parseAddress(address)
	if err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
		return core.EndResult(r)
	}
	switch v := value.(type) {
	case bool:
		err = c.writeCoil(addr, v)
	case int:
		err = c.writeInt32(addr, int32(v))
	case uint:
		err = c.writeUint32(addr, uint32(v))
	case int16:
		err = c.writeRegisters(addr, []uint16{uint16(v)})
	case uint16:
		err = c.writeRegisters(addr, []uint16{v})
	case int32:
		err = c.writeInt32(addr, v)
	case uint32:
		err = c.writeUint32(addr, v)
	case int64:
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], uint64(v))
		err = c.writeRegisters(addr, bytesToRegisters(b[:]))
	case uint64:
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], v)
		err = c.writeRegisters(addr, bytesToRegisters(b[:]))
	case float32:
		err = c.writeUint32(addr, math.Float32bits(v))
	case float64:
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], math.Float64bits(v))
		err = c.writeRegisters(addr, bytesToRegisters(b[:]))
	case string:
		err = c.writeRegisters(addr, bytesToRegisters([]byte(v)))
	case []byte:
		err = c.writeRegisters(addr, bytesToRegisters(v))
	default:
		err = fmt.Errorf("unsupported write type %T", value)
	}
	if err != nil {
		r.IsSucceed = false
		r.Err = err.Error()
	}
	return core.EndResult(r)
}

func (c *modbusClient) readRegisters(address, count uint16, functionCode byte) ([]uint16, error) {
	payload := make([]byte, 5)
	payload[0] = functionCode
	binary.BigEndian.PutUint16(payload[1:3], address)
	binary.BigEndian.PutUint16(payload[3:5], count)
	resp, err := c.sendPDU(payload)
	if err != nil {
		return nil, err
	}
	if len(resp) < 2 || resp[0] != functionCode {
		return nil, fmt.Errorf("invalid function code response")
	}
	byteCount := int(resp[1])
	if len(resp) != byteCount+2 || byteCount != int(count)*2 {
		return nil, fmt.Errorf("invalid register byte count")
	}
	out := make([]uint16, count)
	for i := 0; i < int(count); i++ {
		out[i] = binary.BigEndian.Uint16(resp[2+i*2 : 4+i*2])
	}
	return out, nil
}

func (c *modbusClient) writeRegisters(address uint16, values []uint16) error {
	if len(values) == 0 {
		return nil
	}
	if len(values) > 123 {
		return fmt.Errorf("too many registers: %d", len(values))
	}
	payload := make([]byte, 6+len(values)*2)
	payload[0] = 0x10
	binary.BigEndian.PutUint16(payload[1:3], address)
	binary.BigEndian.PutUint16(payload[3:5], uint16(len(values)))
	payload[5] = byte(len(values) * 2)
	for i, v := range values {
		binary.BigEndian.PutUint16(payload[6+i*2:8+i*2], v)
	}
	resp, err := c.sendPDU(payload)
	if err != nil {
		return err
	}
	if len(resp) != 5 || resp[0] != 0x10 {
		return fmt.Errorf("invalid write register response")
	}
	return nil
}

func (c *modbusClient) writeCoil(address uint16, value bool) error {
	payload := make([]byte, 5)
	payload[0] = 0x05
	binary.BigEndian.PutUint16(payload[1:3], address)
	if value {
		binary.BigEndian.PutUint16(payload[3:5], 0xFF00)
	}
	resp, err := c.sendPDU(payload)
	if err != nil {
		return err
	}
	if len(resp) != 5 || resp[0] != 0x05 {
		return fmt.Errorf("invalid write coil response")
	}
	return nil
}

func (c *modbusClient) sendPDU(pdu []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	policy := c.retry.Normalize()
	var lastErr error
	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		out, err := c.sendPDUOnceLocked(pdu)
		if err == nil {
			if c.autoClose {
				_ = c.closeLocked()
			}
			return out, nil
		}
		lastErr = err
		_ = c.closeLocked()
		if attempt < policy.MaxRetries {
			time.Sleep(policy.RetryDelay)
		}
	}
	return nil, fmt.Errorf("modbus request failed after %d attempts: %w", policy.MaxRetries+1, lastErr)
}

func (c *modbusClient) sendPDUOnceLocked(pdu []byte) ([]byte, error) {
	if err := c.ensureConnLocked(); err != nil {
		return nil, err
	}
	if ds, ok := c.conn.(deadlineSetter); ok {
		if err := ds.SetDeadline(time.Now().Add(c.timeout)); err != nil {
			return nil, err
		}
	}
	switch c.mode {
	case modeTCP:
		return c.sendTCPFrameLocked(pdu)
	case modeRTUOverTCP, modeRTU:
		return c.sendRTUFrameLocked(pdu)
	case modeASCII:
		return c.sendASCIIFrameLocked(pdu)
	default:
		return nil, fmt.Errorf("unsupported mode")
	}
}

func (c *modbusClient) sendTCPFrameLocked(pdu []byte) ([]byte, error) {
	c.transID++
	header := make([]byte, 7)
	binary.BigEndian.PutUint16(header[0:2], c.transID)
	// protocol id is 0
	binary.BigEndian.PutUint16(header[4:6], uint16(len(pdu)+1))
	header[6] = c.unitID
	frame := append(header, pdu...)
	if _, err := c.conn.Write(frame); err != nil {
		return nil, err
	}
	respHead := make([]byte, 7)
	if _, err := io.ReadFull(c.conn, respHead); err != nil {
		return nil, err
	}
	if binary.BigEndian.Uint16(respHead[0:2]) != c.transID {
		return nil, fmt.Errorf("transaction id mismatch")
	}
	if respHead[6] != c.unitID {
		return nil, fmt.Errorf("unit id mismatch")
	}
	bodyLen := int(binary.BigEndian.Uint16(respHead[4:6])) - 1
	if bodyLen <= 0 {
		return nil, fmt.Errorf("invalid response length")
	}
	resp := make([]byte, bodyLen)
	if _, err := io.ReadFull(c.conn, resp); err != nil {
		return nil, err
	}
	if len(resp) >= 2 && resp[0]&0x80 == 0x80 {
		return nil, fmt.Errorf("modbus exception code: 0x%02X", resp[1])
	}
	return resp, nil
}

func (c *modbusClient) sendRTUFrameLocked(pdu []byte) ([]byte, error) {
	frame := make([]byte, 0, len(pdu)+3)
	frame = append(frame, c.unitID)
	frame = append(frame, pdu...)
	crc := crc16(frame)
	frame = append(frame, byte(crc&0xFF), byte(crc>>8))
	if _, err := c.conn.Write(frame); err != nil {
		return nil, err
	}
	response, err := readRTUResponse(c.conn)
	if err != nil {
		return nil, err
	}
	if len(response) < 4 {
		return nil, fmt.Errorf("response too short")
	}
	if response[0] != c.unitID {
		return nil, fmt.Errorf("unit id mismatch")
	}
	gotCRC := binary.LittleEndian.Uint16(response[len(response)-2:])
	if crc16(response[:len(response)-2]) != gotCRC {
		return nil, fmt.Errorf("crc check failed")
	}
	pduResp := response[1 : len(response)-2]
	if len(pduResp) >= 2 && pduResp[0]&0x80 == 0x80 {
		return nil, fmt.Errorf("modbus exception code: 0x%02X", pduResp[1])
	}
	return pduResp, nil
}

func (c *modbusClient) sendASCIIFrameLocked(pdu []byte) ([]byte, error) {
	raw := make([]byte, 0, len(pdu)+2)
	raw = append(raw, c.unitID)
	raw = append(raw, pdu...)
	raw = append(raw, lrc(raw))
	frame := make([]byte, 0, 1+len(raw)*2+2)
	frame = append(frame, ':')
	hexBody := make([]byte, hex.EncodedLen(len(raw)))
	hex.Encode(hexBody, raw)
	frame = append(frame, bytes.ToUpper(hexBody)...)
	frame = append(frame, '\r', '\n')
	if _, err := c.conn.Write(frame); err != nil {
		return nil, err
	}
	respLine, err := readASCIIResponseLine(c.conn)
	if err != nil {
		return nil, err
	}
	if len(respLine) < 1 || respLine[0] != ':' {
		return nil, fmt.Errorf("ascii response missing ':'")
	}
	trimmed := strings.TrimSpace(string(respLine[1:]))
	if len(trimmed)%2 != 0 {
		return nil, fmt.Errorf("ascii response hex length invalid")
	}
	rawResp, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("ascii response decode failed: %w", err)
	}
	if len(rawResp) < 3 {
		return nil, fmt.Errorf("ascii response too short")
	}
	if rawResp[0] != c.unitID {
		return nil, fmt.Errorf("unit id mismatch")
	}
	data := rawResp[:len(rawResp)-1]
	if lrc(data) != rawResp[len(rawResp)-1] {
		return nil, fmt.Errorf("lrc check failed")
	}
	pduResp := data[1:]
	if len(pduResp) >= 2 && pduResp[0]&0x80 == 0x80 {
		return nil, fmt.Errorf("modbus exception code: 0x%02X", pduResp[1])
	}
	return pduResp, nil
}

func (c *modbusClient) ensureConnLocked() error {
	if c.conn != nil {
		return nil
	}
	dial := c.dial
	if dial == nil {
		dial = tcpDial
	}
	conn, err := dial(c)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *modbusClient) closeLocked() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func parseAddress(address string) (uint16, error) {
	addr := strings.TrimSpace(address)
	if addr == "" {
		return 0, core.ErrInvalidAddress
	}
	n, err := strconv.ParseUint(addr, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", core.ErrInvalidAddress, address)
	}
	return uint16(n), nil
}

func bytesToRegisters(values []byte) []uint16 {
	if len(values)%2 != 0 {
		values = append(values, 0x00)
	}
	out := make([]uint16, len(values)/2)
	for i := 0; i < len(out); i++ {
		out[i] = binary.BigEndian.Uint16(values[i*2 : i*2+2])
	}
	return out
}

func registersToBytes(values []uint16) []byte {
	out := make([]byte, len(values)*2)
	for i, v := range values {
		binary.BigEndian.PutUint16(out[i*2:i*2+2], v)
	}
	return out
}

func crc16(data []byte) uint16 {
	var crc uint16 = 0xFFFF
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func lrc(data []byte) byte {
	var sum byte
	for _, b := range data {
		sum += b
	}
	return byte((^sum) + 1)
}

func readRTUResponse(r io.Reader) ([]byte, error) {
	first := make([]byte, 2)
	if _, err := io.ReadFull(r, first); err != nil {
		return nil, err
	}
	function := first[1]
	var response []byte
	if function&0x80 == 0x80 {
		rest := make([]byte, 3)
		if _, err := io.ReadFull(r, rest); err != nil {
			return nil, err
		}
		response = append(first, rest...)
		return response, nil
	}
	switch function {
	case 0x01, 0x02, 0x03, 0x04:
		count := make([]byte, 1)
		if _, err := io.ReadFull(r, count); err != nil {
			return nil, err
		}
		rest := make([]byte, int(count[0])+2)
		if _, err := io.ReadFull(r, rest); err != nil {
			return nil, err
		}
		response = append(response, first...)
		response = append(response, count[0])
		response = append(response, rest...)
	case 0x05, 0x06, 0x0F, 0x10:
		rest := make([]byte, 6)
		if _, err := io.ReadFull(r, rest); err != nil {
			return nil, err
		}
		response = append(first, rest...)
	default:
		return nil, fmt.Errorf("unsupported function code response: 0x%02X", function)
	}
	return response, nil
}

func readASCIIResponseLine(r io.Reader) ([]byte, error) {
	var out []byte
	one := make([]byte, 1)
	for len(out) < 1024 {
		if _, err := io.ReadFull(r, one); err != nil {
			return nil, err
		}
		out = append(out, one[0])
		if one[0] == '\n' {
			return out, nil
		}
	}
	return nil, fmt.Errorf("ascii response too long")
}

func (c *modbusClient) writeInt32(addr uint16, v int32) error {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(v))
	return c.writeRegisters(addr, bytesToRegisters(b[:]))
}

func (c *modbusClient) writeUint32(addr uint16, v uint32) error {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	return c.writeRegisters(addr, bytesToRegisters(b[:]))
}

func tcpDial(c *modbusClient) (streamConn, error) {
	return net.DialTimeout("tcp", c.endpoint, c.timeout)
}

func serialDial(c *modbusClient) (streamConn, error) {
	p, err := serial.Open(c.endpoint, &c.serialMode)
	if err != nil {
		return nil, err
	}
	_ = p.SetReadTimeout(c.timeout)
	return p, nil
}

func unsupported(op string) core.Result {
	r := core.NewResult()
	r.IsSucceed = false
	r.Err = fmt.Sprintf("%s: %v", op, core.ErrUnsupported)
	return core.EndResult(r)
}

func unsupportedT[T any](op string) core.ResultT[T] {
	r := core.ResultT[T]{Result: core.NewResult()}
	r.IsSucceed = false
	r.Err = fmt.Sprintf("%s: %v", op, core.ErrUnsupported)
	return core.EndResultT(r)
}
