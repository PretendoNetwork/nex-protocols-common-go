package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
)

func (commonProtocol *CommonProtocol) acquireNexUniqueID(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pNexUniqueID, nexError := commonProtocol.manager.GenerateNEXUniqueID(commonProtocol.manager, packet.Sender().PID())
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pNexUniqueID.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodAcquireNexUniqueID
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
