package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func unregisterGathering(err error, packet nex.PacketInterface, callID uint32, idGathering uint32) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonMatchMakingProtocol.server
	client := packet.Sender().(*nex.PRUDPClient)

	var session *common_globals.CommonMatchmakeSession
	var ok bool
	if session, ok = common_globals.Sessions[idGathering]; !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	if session.GameMatchmakeSession.Gathering.OwnerPID != client.PID().LegacyValue() {
		return nex.Errors.RendezVous.PermissionDenied
	}

	gatheringPlayers := session.ConnectionIDs

	delete(common_globals.Sessions, idGathering)

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteBool(true) // %retval%

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodUnregisterGathering
	rmcResponse.CallID = callID

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if server.PRUDPVersion == 0 {
		responsePacket, _ = nex.NewPRUDPPacketV0(client, nil)
	} else {
		responsePacket, _ = nex.NewPRUDPPacketV1(client, nil)
	}

	responsePacket.SetType(nex.DataPacket)
	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)
	responsePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	responsePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	responsePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	responsePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	responsePacket.SetPayload(rmcResponseBytes)

	server.Send(responsePacket)

	category := notifications.NotificationCategories.GatheringUnregistered
	subtype := notifications.NotificationSubTypes.GatheringUnregistered.None

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = client.PID().LegacyValue()
	oEvent.Type = notifications.BuildNotificationType(category, subtype)
	oEvent.Param1 = idGathering

	stream := nex.NewStreamOut(server)
	oEventBytes := oEvent.Bytes(stream)

	rmcRequest := nex.NewRMCRequest()
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.CallID = common_globals.CurrentMatchmakingCallID.Next()
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.Parameters = oEventBytes

	rmcRequestBytes := rmcRequest.Bytes()

	for _, connectionID := range gatheringPlayers {
		targetClient := server.FindClientByConnectionID(connectionID)
		if targetClient != nil {
			var messagePacket nex.PRUDPPacketInterface

			if server.PRUDPVersion == 0 {
				messagePacket, _ = nex.NewPRUDPPacketV0(targetClient, nil)
			} else {
				messagePacket, _ = nex.NewPRUDPPacketV1(targetClient, nil)
			}

			messagePacket.SetType(nex.DataPacket)
			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)
			messagePacket.SetSourceStreamType(client.DestinationStreamType)
			messagePacket.SetSourcePort(client.DestinationPort)
			messagePacket.SetDestinationStreamType(client.SourceStreamType)
			messagePacket.SetDestinationPort(client.SourcePort)
			messagePacket.SetPayload(rmcRequestBytes)

			server.Send(messagePacket)
		} else {
			common_globals.Logger.Warning("Client not found")
		}
	}

	return 0
}
