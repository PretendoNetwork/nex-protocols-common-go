package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func autoMatchmakeWithSearchCriteria_Postpone(err error, packet nex.PacketInterface, callID uint32, lstSearchCriteria []*match_making_types.MatchmakeSessionSearchCriteria, anyGathering *nex.DataHolder, message string) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonMatchmakeExtensionProtocol.server
	client := packet.Sender()

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	common_globals.RemoveClientFromAllSessions(client)

	var matchmakeSession *match_making_types.MatchmakeSession
	anyGatheringDataType := anyGathering.TypeName()

	if anyGatheringDataType == "MatchmakeSession" {
		matchmakeSession = anyGathering.ObjectData().(*match_making_types.MatchmakeSession)
	} else {
		common_globals.Logger.Critical("Non-MatchmakeSession DataType?!")
		return nex.Errors.Core.InvalidArgument
	}

	sessions := common_globals.FindSessionsByMatchmakeSessionSearchCriterias(client.PID(), lstSearchCriteria, commonMatchmakeExtensionProtocol.gameSpecificMatchmakeSessionSearchCriteriaChecksHandler)
	var session *common_globals.CommonMatchmakeSession

	if len(sessions) == 0 {
		var errCode uint32
		session, err, errCode = common_globals.CreateSessionByMatchmakeSession(matchmakeSession, nil, client.PID())
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return errCode
		}
	} else {
		session = sessions[0]
	}

	err, errCode := common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID()}, client, message)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return errCode
	}

	rmcResponseStream := nex.NewStreamOut(server)
	matchmakeDataHolder := nex.NewDataHolder()
	matchmakeDataHolder.SetTypeName("MatchmakeSession")
	matchmakeDataHolder.SetObjectData(session.GameMatchmakeSession)
	rmcResponseStream.WriteDataHolder(matchmakeDataHolder)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodAutoMatchmakeWithSearchCriteriaPostpone, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(packet.Destination())
	responsePacket.SetDestination(packet.Source())
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	return 0
}
