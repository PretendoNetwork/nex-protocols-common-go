package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func unregisterGathering(err error, client *nex.Client, callID uint32, idGathering uint32) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonMatchMakingProtocol.server

	var session *common_globals.CommonMatchmakeSession
	var ok bool
	if session, ok = common_globals.Sessions[idGathering]; !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	if session.GameMatchmakeSession.Gathering.OwnerPID != client.PID() {
		return nex.Errors.RendezVous.PermissionDenied
	}

	gatheringPlayers := session.ConnectionIDs

	delete(common_globals.Sessions, idGathering)

	rmcResponse := nex.NewRMCResponse(match_making.ProtocolID, callID)
	rmcResponse.SetSuccess(match_making.MethodUnregisterGathering, nil)

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

	rmcMessage := nex.NewRMCRequest()
	rmcMessage.SetProtocolID(notifications.ProtocolID)
	rmcMessage.SetCallID(common_globals.CurrentMatchmakingCallID.Increment())
	rmcMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

	category := notifications.NotificationCategories.GatheringUnregistered
	subtype := notifications.NotificationSubTypes.GatheringUnregistered.None

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = client.PID()
	oEvent.Type = notifications.BuildNotificationType(category, subtype)
	oEvent.Param1 = idGathering

	stream := nex.NewStreamOut(server)
	oEventBytes := oEvent.Bytes(stream)
	rmcMessage.SetParameters(oEventBytes)

	rmcRequestBytes := rmcMessage.Bytes()

	for _, connectionID := range gatheringPlayers {
		targetClient := server.FindClientFromConnectionID(connectionID)
		if targetClient != nil {
			var messagePacket nex.PacketInterface

			if server.PRUDPVersion() == 0 {
				messagePacket, _ = nex.NewPacketV0(targetClient, nil)
				messagePacket.SetVersion(0)
			} else {
				messagePacket, _ = nex.NewPacketV1(targetClient, nil)
				messagePacket.SetVersion(1)
			}

			messagePacket.SetSource(0xA1)
			messagePacket.SetDestination(0xAF)
			messagePacket.SetType(nex.DataPacket)
			messagePacket.SetPayload(rmcRequestBytes)

			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)

			server.Send(messagePacket)
		} else {
			logger.Warning("Client not found")
		}
	}

	return 0
}
