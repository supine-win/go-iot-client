package plc

import (
	"fmt"

	"github.com/example/go-iotclient/core"
)

type SiemensClient struct{ endpoint string }
type OmronFinsClient struct{ endpoint string }
type AllenBradleyClient struct{ endpoint string }

func NewSiemensClient(endpoint string) *SiemensClient               { return &SiemensClient{endpoint: endpoint} }
func NewOmronFinsClient(endpoint string) *OmronFinsClient           { return &OmronFinsClient{endpoint: endpoint} }
func NewAllenBradleyClient(endpoint string) *AllenBradleyClient     { return &AllenBradleyClient{endpoint: endpoint} }

func (c *SiemensClient) Open() core.Result          { return unsupported("SiemensClient.Open") }
func (c *SiemensClient) Close() core.Result         { return core.EndResult(core.NewResult()) }
func (c *SiemensClient) Connected() bool            { return false }
func (c *SiemensClient) ReadInt16(string) core.ResultT[int16] { return unsupportedT[int16]("SiemensClient.ReadInt16") }
func (c *SiemensClient) ReadInt32(string) core.ResultT[int32] { return unsupportedT[int32]("SiemensClient.ReadInt32") }
func (c *SiemensClient) ReadFloat(string) core.ResultT[float32] { return unsupportedT[float32]("SiemensClient.ReadFloat") }
func (c *SiemensClient) ReadString(string, int) core.ResultT[string] { return unsupportedT[string]("SiemensClient.ReadString") }
func (c *SiemensClient) Write(string, interface{}) core.Result { return unsupported("SiemensClient.Write") }

func (c *OmronFinsClient) Open() core.Result          { return unsupported("OmronFinsClient.Open") }
func (c *OmronFinsClient) Close() core.Result         { return core.EndResult(core.NewResult()) }
func (c *OmronFinsClient) Connected() bool            { return false }
func (c *OmronFinsClient) ReadInt16(string) core.ResultT[int16] { return unsupportedT[int16]("OmronFinsClient.ReadInt16") }
func (c *OmronFinsClient) ReadInt32(string) core.ResultT[int32] { return unsupportedT[int32]("OmronFinsClient.ReadInt32") }
func (c *OmronFinsClient) ReadFloat(string) core.ResultT[float32] { return unsupportedT[float32]("OmronFinsClient.ReadFloat") }
func (c *OmronFinsClient) ReadString(string, int) core.ResultT[string] { return unsupportedT[string]("OmronFinsClient.ReadString") }
func (c *OmronFinsClient) Write(string, interface{}) core.Result { return unsupported("OmronFinsClient.Write") }

func (c *AllenBradleyClient) Open() core.Result          { return unsupported("AllenBradleyClient.Open") }
func (c *AllenBradleyClient) Close() core.Result         { return core.EndResult(core.NewResult()) }
func (c *AllenBradleyClient) Connected() bool            { return false }
func (c *AllenBradleyClient) ReadInt16(string) core.ResultT[int16] { return unsupportedT[int16]("AllenBradleyClient.ReadInt16") }
func (c *AllenBradleyClient) ReadInt32(string) core.ResultT[int32] { return unsupportedT[int32]("AllenBradleyClient.ReadInt32") }
func (c *AllenBradleyClient) ReadFloat(string) core.ResultT[float32] { return unsupportedT[float32]("AllenBradleyClient.ReadFloat") }
func (c *AllenBradleyClient) ReadString(string, int) core.ResultT[string] { return unsupportedT[string]("AllenBradleyClient.ReadString") }
func (c *AllenBradleyClient) Write(string, interface{}) core.Result { return unsupported("AllenBradleyClient.Write") }

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

