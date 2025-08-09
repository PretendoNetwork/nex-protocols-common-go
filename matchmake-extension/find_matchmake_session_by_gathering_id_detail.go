package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) findMatchmakeSessionByGatheringIDDetail(err error, packet nex.PacketInterface, callID uint32, gid types.UInt32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.RLock()

	// * Only participants can call this method
	nexError := database.CheckGatheringForParticipation(commonProtocol.manager, uint32(gid), []types.PID{connection.PID()})
	if nexError != nil {
		commonProtocol.manager.Mutex.RUnlock()
		return nil, nexError
	}

	matchmakeSession, _, nexError := database.GetMatchmakeSessionByID(commonProtocol.manager, endpoint, uint32(gid))
	if nexError != nil {
		commonProtocol.manager.Mutex.RUnlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.RUnlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	matchmakeSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodFindMatchmakeSessionByGatheringIDDetail
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterFindMatchmakeSessionByGatheringIDDetail != nil {
		go commonProtocol.OnAfterFindMatchmakeSessionByGatheringIDDetail(packet, gid)
	}

	return rmcResponse, nil
}
