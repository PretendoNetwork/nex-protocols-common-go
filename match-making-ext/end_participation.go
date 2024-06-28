package match_making_ext

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/v2/match-making-ext"
)

func (commonProtocol *CommonProtocol) endParticipation(err error, packet nex.PacketInterface, callID uint32, idGathering *types.PrimitiveU32, strMessage *types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	common_globals.MatchmakingMutex.Lock()

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	nexError := database.EndGatheringParticipation(commonProtocol.db, idGathering.Value, connection, strMessage.Value)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	common_globals.MatchmakingMutex.Unlock()

	retval := types.NewPrimitiveBool(true)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = match_making_ext.ProtocolID
	rmcResponse.MethodID = match_making_ext.MethodEndParticipation
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterEndParticipation != nil {
		go commonProtocol.OnAfterEndParticipation(packet, idGathering, strMessage)
	}

	return rmcResponse, nil
}
