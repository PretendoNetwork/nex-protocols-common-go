package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility/database"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

func (commonProtocol *CommonProtocol) associateNexUniqueIDWithMyPrincipalID(err error, packet nex.PacketInterface, callID uint32, uniqueIdInfo utility_types.UniqueIDInfo) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if uniqueIdInfo.NEXUniqueID == 0 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "No unique id sent")
	}

	nexError := utility_database.CheckCanAssociateUniqueIDs(commonProtocol.manager, packet.Sender().PID(), []uint64{uint64(uniqueIdInfo.NEXUniqueID)}, []uint64{uint64(uniqueIdInfo.NEXUniqueIDPassword)})
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	nexError = utility_database.ClearPIDUniqueIDAssociations(commonProtocol.manager, packet.Sender().PID())
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	if uniqueIdInfo.NEXUniqueIDPassword == types.UInt64(0) {
		nexError = utility_database.InsertUniqueIDsByUser(commonProtocol.manager, packet.Sender().PID(), []uint64{uint64(uniqueIdInfo.NEXUniqueID)}, true)
	} else {
		nexError = utility_database.InsertUniqueIDsByUserWithPasswords(commonProtocol.manager, packet.Sender().PID(), []uint64{uint64(uniqueIdInfo.NEXUniqueID)}, []uint64{uint64(uniqueIdInfo.NEXUniqueIDPassword)}, true)
	}

	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodAssociateNexUniqueIDWithMyPrincipalID
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
