package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) updateProgressScore(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32, progressScore *types.PrimitiveU8) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	if progressScore.Value > 100 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	common_globals.MatchmakingMutex.Lock()

	session, _, nexError := database.GetMatchmakeSessionByID(commonProtocol.db, endpoint, gid.Value)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	if !session.Gathering.OwnerPID.Equals(connection.PID()) {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	nexError = database.UpdateProgressScore(commonProtocol.db, gid.Value, progressScore.Value)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	common_globals.MatchmakingMutex.Unlock()

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodUpdateProgressScore
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterUpdateProgressScore != nil {
		go commonProtocol.OnAfterUpdateProgressScore(packet, gid, progressScore)
	}

	return rmcResponse, nil
}
