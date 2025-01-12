package matchmake_extension

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) getSimplePlayingSession(err error, packet nex.PacketInterface, callID uint32, listPID types.List[types.PID], includeLoginUser types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	// * Does nothing if element is not present in the List
	listPID = slices.DeleteFunc(listPID, func(pid types.PID) bool {
		return pid == connection.PID()
	})

	if includeLoginUser {
		listPID = append(listPID, connection.PID())
	}

	commonProtocol.manager.Mutex.RLock()

	simplePlayingSessions, nexError := database.GetSimplePlayingSession(commonProtocol.manager, listPID)
	if nexError != nil {
		commonProtocol.manager.Mutex.RUnlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.RUnlock()

	lstSimplePlayingSession := types.NewList[match_making_types.SimplePlayingSession]()
	lstSimplePlayingSession = simplePlayingSessions

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	lstSimplePlayingSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodGetSimplePlayingSession
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterGetSimplePlayingSession != nil {
		go commonProtocol.OnAfterGetSimplePlayingSession(packet, listPID, includeLoginUser)
	}

	return rmcResponse, nil
}
