package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
)

func (commonProtocol *CommonProtocol) getPlayingSession(err error, packet nex.PacketInterface, callID uint32, lstPID types.List[types.PID]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	if len(lstPID) > 300 { // * MAX_PRINCIPALID_SIZE_TO_FIND_MATCHMAKE_SESSION
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "lstPID is bigger than MAX_PRINCIPALID_SIZE_TO_FIND_MATCHMAKE_SESSION")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.RLock()

	lstPlayingSession, nexError := database.GetPlayingSession(commonProtocol.manager, lstPID)
	if nexError != nil {
		commonProtocol.manager.Mutex.RUnlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.RUnlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	lstPlayingSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodGetPlayingSession
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterGetPlayingSession != nil {
		go commonProtocol.OnAfterGetPlayingSession(packet, lstPID)
	}

	return rmcResponse, nil
}
