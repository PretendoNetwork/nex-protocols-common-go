package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility/database"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

func (commonProtocol *CommonProtocol) getAssociatedNexUniqueIdWithMyPrincipalId(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	uniqueIds, passwords, nexError := utility_database.GetUserAssociatedUniqueIDs(commonProtocol.manager, packet.Sender().PID())
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	if len(uniqueIds) == 0 || len(passwords) == 0 {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, "No unique id for this user")
	}

	uniqueIdInfo := utility_types.NewUniqueIDInfo()
	uniqueIdInfo.NEXUniqueID = types.UInt64(uniqueIds[0])
	uniqueIdInfo.NEXUniqueIDPassword = types.UInt64(passwords[0])

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	uniqueIdInfo.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodGetAssociatedNexUniqueIDWithMyPrincipalID
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
