package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
)

func (commonProtocol *CommonProtocol) getStringSettings(err error, packet nex.PacketInterface, callID uint32, stringSettingIndex types.UInt32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if commonProtocol.manager.GetStringSettings == nil {
		common_globals.Logger.Warning("GetStringSettings not defined")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	stringSettings, nexError := commonProtocol.manager.GetStringSettings(commonProtocol.manager, packet.Sender().PID(), stringSettingIndex)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	stringSettings.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodGetStringSettings
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
