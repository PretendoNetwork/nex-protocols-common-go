package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func browseMatchmakeSession(err error, client *nex.Client, callID uint32, searchCriteria *match_making_types.MatchmakeSessionSearchCriteria, resultRange *nex.ResultRange) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonMatchmakeExtensionProtocol.server
	searchCriterias := []*match_making_types.MatchmakeSessionSearchCriteria{searchCriteria}

	sessions := common_globals.FindSessionsByMatchmakeSessionSearchCriterias(client.PID(), searchCriterias, commonMatchmakeExtensionProtocol.gameSpecificMatchmakeSessionSearchCriteriaChecksHandler)

	if len(sessions) < int(resultRange.Offset) {
		return nex.Errors.Core.InvalidIndex
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

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodBrowseMatchmakeSession, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	return 0
}
