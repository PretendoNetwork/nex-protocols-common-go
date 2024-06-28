package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) modifyCurrentGameAttribute(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32, attribIndex *types.PrimitiveU32, newValue *types.PrimitiveU32) (*nex.RMCMessage, *nex.Error) {
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

	index := int(attribIndex.Value)

	if index >= session.Attributes.Length() {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidIndex, "change_error")
	}

	nexError = database.UpdateGameAttribute(commonProtocol.db, gid.Value, attribIndex.Value, newValue.Value)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	common_globals.MatchmakingMutex.Unlock()

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodModifyCurrentGameAttribute
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterModifyCurrentGameAttribute != nil {
		go commonProtocol.OnAfterModifyCurrentGameAttribute(packet, gid, attribIndex, newValue)
	}

	return rmcResponse, nil
}
