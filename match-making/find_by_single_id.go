package matchmaking

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	matchmake_extension_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
)

func (commonProtocol *CommonProtocol) findBySingleID(err error, packet nex.PacketInterface, callID uint32, id *types.PrimitiveU32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	common_globals.MatchmakingMutex.RLock()
	gathering, gatheringType, participants, startedTime, nexError := database.FindGatheringByID(commonProtocol.db, id.Value)
	if nexError != nil {
		common_globals.MatchmakingMutex.RUnlock()
		return nil, nexError
	}

	// TODO - Add PersistentGathering
	if gatheringType != "MatchmakeSession" {
		common_globals.MatchmakingMutex.RUnlock()
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	matchmakeSession, nexError := matchmake_extension_database.GetMatchmakeSessionByGathering(commonProtocol.db, endpoint, gathering, uint32(len(participants)), startedTime)
	if nexError != nil {
		common_globals.MatchmakingMutex.RUnlock()
		return nil, nexError
	}

	common_globals.MatchmakingMutex.RUnlock()

	bResult := types.NewPrimitiveBool(true)
	pGathering := types.NewAnyDataHolder()

	pGathering.TypeName = types.NewString(gatheringType)
	pGathering.ObjectData = matchmakeSession.Copy()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	bResult.WriteTo(rmcResponseStream)
	pGathering.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodFindBySingleID
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterFindBySingleID != nil {
		go commonProtocol.OnAfterFindBySingleID(packet, id)
	}

	return rmcResponse, nil
}
