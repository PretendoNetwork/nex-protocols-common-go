package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func createMatchmakeSessionWithParam(err error, client *nex.Client, callID uint32, createMatchmakeSessionParam *match_making_types.CreateMatchmakeSessionParam) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := client.Server()

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from all sessions
	common_globals.RemoveConnectionIDFromAllSessions(client.ConnectionID())

	joinedMatchmakeSession := createMatchmakeSessionParam.SourceMatchmakeSession.Copy().(*match_making_types.MatchmakeSession)

	sessionIndex := common_globals.GetSessionIndex()
	if sessionIndex == 0 {
		logger.Critical("No gatherings available!")
		return nex.Errors.RendezVous.LimitExceeded
	}

	joinedMatchmakeSession.SetStructureVersion(3)
	joinedMatchmakeSession.Gathering.ID = sessionIndex
	joinedMatchmakeSession.Gathering.OwnerPID = client.PID()
	joinedMatchmakeSession.Gathering.HostPID = client.PID()
	joinedMatchmakeSession.StartedTime = nex.NewDateTime(0)
	joinedMatchmakeSession.StartedTime.UTC()
	joinedMatchmakeSession.SessionKey = make([]byte, 32)

	// TODO - Are these parameters game-specific?

	joinedMatchmakeSession.MatchmakeParam.Parameters["@SR"] = nex.NewVariant()
	joinedMatchmakeSession.MatchmakeParam.Parameters["@SR"].TypeID = 3
	joinedMatchmakeSession.MatchmakeParam.Parameters["@SR"].Bool = true

	joinedMatchmakeSession.MatchmakeParam.Parameters["@GIR"] = nex.NewVariant()
	joinedMatchmakeSession.MatchmakeParam.Parameters["@GIR"].TypeID = 1
	joinedMatchmakeSession.MatchmakeParam.Parameters["@GIR"].Int64 = 3

	common_globals.Sessions[sessionIndex] = &common_globals.CommonMatchmakeSession{
		SearchCriteria:       make([]*match_making_types.MatchmakeSessionSearchCriteria, 0),
		GameMatchmakeSession: joinedMatchmakeSession,
	}

	err, errCode := common_globals.AddPlayersToSession(common_globals.Sessions[sessionIndex], []uint32{client.ConnectionID()})
	if err != nil {
		logger.Error(err.Error())
		return errCode
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteStructure(joinedMatchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodCreateMatchmakeSessionWithParam, rmcResponseBody)

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

	// * Works for Minecraft, not tried on anything else
	notificationRequestMessage := nex.NewRMCRequest()
	notificationRequestMessage.SetProtocolID(notifications.ProtocolID)
	notificationRequestMessage.SetCallID(0xffff0000 + callID)
	notificationRequestMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

	notificationCategory := notifications.NotificationCategories.Participation
	notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = client.PID()
	oEvent.Type = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
	oEvent.Param1 = sessionIndex
	oEvent.Param2 = client.PID()
	oEvent.StrParam = ""
	oEvent.Param3 = 1

	notificationStream := nex.NewStreamOut(server)

	notificationStream.WriteStructure(oEvent)

	notificationRequestMessage.SetParameters(notificationStream.Bytes())
	notificationRequestBytes := notificationRequestMessage.Bytes()

	var messagePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
		messagePacket, _ = nex.NewPacketV0(client, nil)
		messagePacket.SetVersion(0)
	} else {
		messagePacket, _ = nex.NewPacketV1(client, nil)
		messagePacket.SetVersion(1)
	}

	messagePacket.SetSource(0xA1)
	messagePacket.SetDestination(0xAF)
	messagePacket.SetType(nex.DataPacket)
	messagePacket.SetPayload(notificationRequestBytes)

	messagePacket.AddFlag(nex.FlagNeedsAck)
	messagePacket.AddFlag(nex.FlagReliable)

	server.Send(messagePacket)

	return 0
}
