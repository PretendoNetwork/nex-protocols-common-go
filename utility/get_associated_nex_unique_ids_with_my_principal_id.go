package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility/database"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

func (commonProtocol *CommonProtocol) getAssociatedNexUniqueIDsWithMyPrincipalID(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	uniqueIDInfos, nexError := utility_database.GetUserAssociatedUniqueIDs(commonProtocol.manager, packet.Sender().PID())
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	if len(uniqueIDInfos) == 0 {
		uniqueIDInfo := utility_types.NewUniqueIDInfo()
		uniqueIDInfo.NEXUniqueID = 0
		uniqueIDInfo.NEXUniqueIDPassword = 0

		uniqueIDInfos = append(uniqueIDInfos, uniqueIDInfo)
	}

	uniqueIDInfos.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodGetAssociatedNexUniqueIDsWithMyPrincipalID
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
