package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

func (commonProtocol *CommonProtocol) acquireNexUniqueIDWithPassword(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pNexUniqueIDInfo := utility_types.NewUniqueIDInfo()

	uniqueId, password, nexError := commonProtocol.manager.GenerateNEXUniqueIDWithPassword(commonProtocol.manager, packet)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	pNexUniqueIDInfo.NEXUniqueID = types.UInt64(uniqueId)
	pNexUniqueIDInfo.NEXUniqueIDPassword = types.UInt64(password)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pNexUniqueIDInfo.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodAcquireNexUniqueIDWithPassword
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
