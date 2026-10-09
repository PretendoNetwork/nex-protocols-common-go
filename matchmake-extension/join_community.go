package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	"github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) JoinCommunity(err error, packet nex.PacketInterface, callID uint32, gid types.UInt32, strMessage types.String, strPassword types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(strMessage) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	commonProtocol.manager.Mutex.Lock()

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	persistentGathering, nexError := database.GetPersistentGatheringByID(commonProtocol.manager, connection.PID(), uint32(gid))
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	count, nexError := database.GetPersistentGatheringParticipationCount(commonProtocol.manager, connection.PID())
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	if count > constants.PersistentGatheringParticipationMax {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PersistentGatheringParticipationMax, "User is participating in too many persistent gatherings")
	}

	if persistentGathering.Password != strPassword {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidPassword, "Wrong password")
	}

	_, nexError = database.JoinPersistentGathering(commonProtocol.manager, persistentGathering, connection, 1, string(strMessage))

	commonProtocol.manager.Mutex.Unlock()

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinCommunity
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
