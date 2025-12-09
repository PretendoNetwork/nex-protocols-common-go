package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility/database"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

func (commonProtocol *CommonProtocol) getAssociatedNexUniqueIdsWithMyPrincipalId(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
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
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, "No unique ids for this user")
	}

	uniqueIdInfos := types.NewList[utility_types.UniqueIDInfo]()

	for _, i := range uniqueIds {
		uniqueIdInfo := utility_types.NewUniqueIDInfo()
		uniqueIdInfo.NEXUniqueID = types.UInt64(uniqueIds[i])
		uniqueIdInfo.NEXUniqueIDPassword = types.UInt64(passwords[i])

		uniqueIdInfos = append(uniqueIdInfos, uniqueIdInfo)
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	uniqueIdInfos.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodGetAssociatedNexUniqueIDsWithMyPrincipalID
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
