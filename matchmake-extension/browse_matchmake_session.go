package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func browseMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, searchCriteria *match_making_types.MatchmakeSessionSearchCriteria, resultRange *types.ResultRange) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	searchCriterias := []*match_making_types.MatchmakeSessionSearchCriteria{searchCriteria}

	sessions := common_globals.FindSessionsByMatchmakeSessionSearchCriterias(connection.PID(), searchCriterias, commonProtocol.GameSpecificMatchmakeSessionSearchCriteriaChecks)

	if len(sessions) < int(resultRange.Offset.Value) {
		return nil, nex.Errors.Core.InvalidIndex
	}

	sessions = sessions[resultRange.Offset.Value:]

	if len(sessions) > int(resultRange.Length.Value) {
		sessions = sessions[:resultRange.Length.Value]
	}

	lstGathering := types.NewList[*types.AnyDataHolder]()
	lstGathering.Type = types.NewAnyDataHolder()

	for _, session := range sessions {
		matchmakeSessionDataHolder := types.NewAnyDataHolder()
		matchmakeSessionDataHolder.TypeName = types.NewString("MatchmakeSession")
		matchmakeSessionDataHolder.ObjectData = session.GameMatchmakeSession.Copy()

		lstGathering.Append(matchmakeSessionDataHolder)
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	lstGathering.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodBrowseMatchmakeSession
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
