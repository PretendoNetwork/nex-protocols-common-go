package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) updateApplicationBuffer(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32, applicationBuffer *types.Buffer) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	common_globals.MatchmakingMutex.Lock()

	session, nexError := database.GetMatchmakeSessionByID(commonProtocol.db, endpoint, gid.Value)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	if !session.Gathering.OwnerPID.Equals(connection.PID()) {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	nexError = database.UpdateApplicationBuffer(commonProtocol.db, gid.Value, applicationBuffer)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	common_globals.MatchmakingMutex.Unlock()

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodUpdateApplicationBuffer
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterUpdateApplicationBuffer != nil {
		go commonProtocol.OnAfterUpdateApplicationBuffer(packet, gid, applicationBuffer)
	}

	return rmcResponse, nil
}
