package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
)

func (commonProtocol *CommonProtocol) getIntegerSettings(err error, packet nex.PacketInterface, callID uint32, integerSettingIndex types.UInt32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if commonProtocol.manager.GetIntegerSettings == nil {
		common_globals.Logger.Warning("GetIntegerSettings not defined")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	integerSettings, nexError := commonProtocol.manager.GetIntegerSettings(commonProtocol.manager, packet, uint32(integerSettingIndex))
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	nexIntegerSettings := make(types.Map[types.UInt16, types.Int32])
	for k, v := range integerSettings {
		nexIntegerSettings[types.UInt16(k)] = types.Int32(v)
	}

	nexIntegerSettings.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodGetIntegerSettings
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
