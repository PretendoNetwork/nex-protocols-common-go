package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) joinCommunity(err error, packet nex.PacketInterface, callID uint32, gid types.UInt32, strMessage types.String, strPassword types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	if len(strMessage) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Message too long")
	}

	commonProtocol.manager.Mutex.Lock()

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	gathering, nexError := database.GetPersistentGatheringByID(commonProtocol.manager, connection.PID(), uint32(gid))
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	if gathering.Password != strPassword {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidPassword, "Wrong password")
	}

	nexError = common_globals.CanJoinCommunity(commonProtocol.manager, connection.PID(), gathering)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	_, nexError = database.JoinPersistentGathering(commonProtocol.manager, gathering, connection, 1, string(strMessage))
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.Unlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinMatchmakeSession
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterJoinCommunity != nil {
		go commonProtocol.OnAfterJoinCommunity(packet, gid, strMessage, strPassword)
	}

	return rmcResponse, nil
}
