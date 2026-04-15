package modbus

import (
	"fmt"

	"github.com/example/go-iotclient/core"
)

type TcpClient struct{ endpoint string }
type RtuOverTcpClient struct{ endpoint string }
type RtuClient struct{ port string }
type AsciiClient struct{ port string }

func NewTcpClient(endpoint string) *TcpClient                 { return &TcpClient{endpoint: endpoint} }
func NewRtuOverTcpClient(endpoint string) *RtuOverTcpClient   { return &RtuOverTcpClient{endpoint: endpoint} }
func NewRtuClient(port string) *RtuClient                     { return &RtuClient{port: port} }
func NewAsciiClient(port string) *AsciiClient                 { return &AsciiClient{port: port} }

func (c *TcpClient) Open() core.Result             { return unsupported("ModbusTcpClient.Open") }
func (c *TcpClient) Close() core.Result            { return core.EndResult(core.NewResult()) }
func (c *TcpClient) Connected() bool               { return false }
func (c *TcpClient) ReadInt16(string) core.ResultT[int16] { return unsupportedT[int16]("ModbusTcpClient.ReadInt16") }
func (c *TcpClient) ReadInt32(string) core.ResultT[int32] { return unsupportedT[int32]("ModbusTcpClient.ReadInt32") }
func (c *TcpClient) ReadFloat(string) core.ResultT[float32] { return unsupportedT[float32]("ModbusTcpClient.ReadFloat") }
func (c *TcpClient) ReadString(string, int) core.ResultT[string] { return unsupportedT[string]("ModbusTcpClient.ReadString") }
func (c *TcpClient) Write(string, interface{}) core.Result { return unsupported("ModbusTcpClient.Write") }

func (c *RtuOverTcpClient) Open() core.Result             { return unsupported("ModbusRtuOverTcpClient.Open") }
func (c *RtuOverTcpClient) Close() core.Result            { return core.EndResult(core.NewResult()) }
func (c *RtuOverTcpClient) Connected() bool               { return false }
func (c *RtuOverTcpClient) ReadInt16(string) core.ResultT[int16] { return unsupportedT[int16]("ModbusRtuOverTcpClient.ReadInt16") }
func (c *RtuOverTcpClient) ReadInt32(string) core.ResultT[int32] { return unsupportedT[int32]("ModbusRtuOverTcpClient.ReadInt32") }
func (c *RtuOverTcpClient) ReadFloat(string) core.ResultT[float32] { return unsupportedT[float32]("ModbusRtuOverTcpClient.ReadFloat") }
func (c *RtuOverTcpClient) ReadString(string, int) core.ResultT[string] { return unsupportedT[string]("ModbusRtuOverTcpClient.ReadString") }
func (c *RtuOverTcpClient) Write(string, interface{}) core.Result { return unsupported("ModbusRtuOverTcpClient.Write") }

func (c *RtuClient) Open() core.Result             { return unsupported("ModbusRtuClient.Open") }
func (c *RtuClient) Close() core.Result            { return core.EndResult(core.NewResult()) }
func (c *RtuClient) Connected() bool               { return false }
func (c *RtuClient) ReadInt16(string) core.ResultT[int16] { return unsupportedT[int16]("ModbusRtuClient.ReadInt16") }
func (c *RtuClient) ReadInt32(string) core.ResultT[int32] { return unsupportedT[int32]("ModbusRtuClient.ReadInt32") }
func (c *RtuClient) ReadFloat(string) core.ResultT[float32] { return unsupportedT[float32]("ModbusRtuClient.ReadFloat") }
func (c *RtuClient) ReadString(string, int) core.ResultT[string] { return unsupportedT[string]("ModbusRtuClient.ReadString") }
func (c *RtuClient) Write(string, interface{}) core.Result { return unsupported("ModbusRtuClient.Write") }

func (c *AsciiClient) Open() core.Result             { return unsupported("ModbusAsciiClient.Open") }
func (c *AsciiClient) Close() core.Result            { return core.EndResult(core.NewResult()) }
func (c *AsciiClient) Connected() bool               { return false }
func (c *AsciiClient) ReadInt16(string) core.ResultT[int16] { return unsupportedT[int16]("ModbusAsciiClient.ReadInt16") }
func (c *AsciiClient) ReadInt32(string) core.ResultT[int32] { return unsupportedT[int32]("ModbusAsciiClient.ReadInt32") }
func (c *AsciiClient) ReadFloat(string) core.ResultT[float32] { return unsupportedT[float32]("ModbusAsciiClient.ReadFloat") }
func (c *AsciiClient) ReadString(string, int) core.ResultT[string] { return unsupportedT[string]("ModbusAsciiClient.ReadString") }
func (c *AsciiClient) Write(string, interface{}) core.Result { return unsupported("ModbusAsciiClient.Write") }

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

