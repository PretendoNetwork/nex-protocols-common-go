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

	uniqueIdList := make([]uint64, 0)
	passwordList := make([]uint64, 0)

	for _, info := range uniqueIdInfos {
		uniqueIdList = append(uniqueIdList, uint64(info.NEXUniqueID))
		passwordList = append(passwordList, uint64(info.NEXUniqueIDPassword))
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

	// The primary difference between with/without passwords is convenience, we can insert zeroed passwords just fine this way
	nexError = utility_database.InsertUniqueIDsByUserWithPasswords(commonProtocol.manager, packet.Sender().PID(), uniqueIdList, passwordList, true)

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
