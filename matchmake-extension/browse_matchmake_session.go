package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func browseMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, searchCriteria *match_making_types.MatchmakeSessionSearchCriteria, resultRange *nex.ResultRange) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	server := commonMatchmakeExtensionProtocol.server
	client := packet.Sender()

	searchCriterias := []*match_making_types.MatchmakeSessionSearchCriteria{searchCriteria}

	sessions := common_globals.FindSessionsByMatchmakeSessionSearchCriterias(client.PID(), searchCriterias, commonMatchmakeExtensionProtocol.gameSpecificMatchmakeSessionSearchCriteriaChecksHandler)

	if len(sessions) < int(resultRange.Offset) {
		return nil, nex.Errors.Core.InvalidIndex
	}

	sessions = sessions[resultRange.Offset:]

	if len(sessions) > int(resultRange.Length) {
		sessions = sessions[:resultRange.Length]
	}

	lstGathering := make([]*nex.DataHolder, len(sessions))

	for _, session := range sessions {
		matchmakeSessionDataHolder := nex.NewDataHolder()
		matchmakeSessionDataHolder.SetTypeName("MatchmakeSession")
		matchmakeSessionDataHolder.SetObjectData(session.GameMatchmakeSession)

		lstGathering = append(lstGathering, matchmakeSessionDataHolder)
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteListDataHolder(lstGathering)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodBrowseMatchmakeSession
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
