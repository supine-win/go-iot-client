package plc

import iotclient "github.com/supine-win/go-iot-client"

type MitsubishiClient = iotclient.MitsubishiClient
type MitsubishiVersion = iotclient.MitsubishiVersion

const (
	MitsubishiVersionNone  = iotclient.MitsubishiVersionNone
	MitsubishiVersionA1E   = iotclient.MitsubishiVersionA1E
	MitsubishiVersionQna3E = iotclient.MitsubishiVersionQna3E
)

func NewMitsubishiClient(version MitsubishiVersion, ip string, port int, timeoutMs int) *MitsubishiClient {
	return iotclient.NewMitsubishiClient(version, ip, port, timeoutMs)
}

