package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility/database"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

func (commonProtocol *CommonProtocol) associateNexUniqueIDsWithMyPrincipalID(err error, packet nex.PacketInterface, callID uint32, uniqueIdInfos types.List[utility_types.UniqueIDInfo]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if len(uniqueIdInfos) == 0 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "No unique ids sent")
	}

	uniqueIdList := make(types.List[types.UInt64], 0)
	passwordList := make(types.List[types.UInt64], 0)

	for _, info := range uniqueIdInfos {
		uniqueIdList = append(uniqueIdList, info.NEXUniqueID)
		passwordList = append(passwordList, info.NEXUniqueIDPassword)
	}

	nexError := utility_database.CheckCanAssociateUniqueIDs(commonProtocol.manager, packet.Sender().PID(), uniqueIdList, passwordList)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	nexError = utility_database.ClearPIDUniqueIDAssociations(commonProtocol.manager, packet.Sender().PID())
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	nexError = utility_database.UpdateUniqueIDAssociations(commonProtocol.manager, packet.Sender().PID(), uniqueIdList, true)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodAssociateNexUniqueIDsWithMyPrincipalID
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
