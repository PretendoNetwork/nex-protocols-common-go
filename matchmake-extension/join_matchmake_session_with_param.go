package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func joinMatchmakeSessionWithParam(err error, client *nex.Client, callID uint32, joinMatchmakeSessionParam *match_making_types.JoinMatchmakeSessionParam) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := client.Server()

	session, ok := common_globals.Sessions[joinMatchmakeSessionParam.GID]
	if !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	// TODO - More checks here

	err, errCode := common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID()})
	if err != nil {
		logger.Error(err.Error())
		return errCode
	}

	joinedMatchmakeSession := session.GameMatchmakeSession

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteStructure(joinedMatchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodJoinMatchmakeSessionWithParam, rmcResponseBody)

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

	for i := 0; i < len(session.ConnectionIDs); i++ {
		target := server.FindClientFromConnectionID(session.ConnectionIDs[i])
		if target == nil {
			// TODO - Error here?
			logger.Warning("Player not found")
			continue
		}

		// * Works for Minecraft, not tried on anything else
		notificationRequestMessage := nex.NewRMCRequest()
		notificationRequestMessage.SetProtocolID(notifications.ProtocolID)
		notificationRequestMessage.SetCallID(0xffff0000 + callID + uint32(i))
		notificationRequestMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

		notificationCategory := notifications.NotificationCategories.Participation
		notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = client.PID()
		oEvent.Type = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
		oEvent.Param1 = joinedMatchmakeSession.ID
		oEvent.Param2 = target.PID()
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
	}

	return 0
}
