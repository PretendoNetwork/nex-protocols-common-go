package match_making_ext

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func endParticipation(err error, packet nex.PacketInterface, callID uint32, idGathering uint32, strMessage string) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonMatchMakingExtProtocol.server
	client := packet.Sender().(*nex.PRUDPClient)

	var session *common_globals.CommonMatchmakeSession
	var ok bool
	if session, ok = common_globals.Sessions[idGathering]; !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	matchmakeSession := session.GameMatchmakeSession
	ownerPID := matchmakeSession.Gathering.OwnerPID

	var deleteSession bool = false
	if client.PID().LegacyValue() == matchmakeSession.Gathering.OwnerPID {
		// This flag tells the server to change the matchmake session owner if they disconnect
		// If the flag is not set, delete the session
		// More info: https://nintendo-wiki.pretendo.network/docs/nex/protocols/match-making/types#flags
		if matchmakeSession.Gathering.Flags&match_making.GatheringFlags.DisconnectChangeOwner == 0 {
			deleteSession = true
		} else {
			common_globals.ChangeSessionOwner(client, idGathering)
		}
	}

	if deleteSession {
		delete(common_globals.Sessions, idGathering)
	} else {
		common_globals.RemoveConnectionIDFromSession(client.ConnectionID, idGathering)
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteBool(true) // %retval%

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = match_making_ext.ProtocolID
	rmcResponse.MethodID = match_making_ext.MethodEndParticipation
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

	category := notifications.NotificationCategories.Participation
	subtype := notifications.NotificationSubTypes.Participation.Ended

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = client.PID().LegacyValue()
	oEvent.Type = notifications.BuildNotificationType(category, subtype)
	oEvent.Param1 = idGathering
	oEvent.Param2 = client.PID().LegacyValue()
	oEvent.StrParam = strMessage

	stream := nex.NewStreamOut(server)
	oEventBytes := oEvent.Bytes(stream)

	rmcRequest := nex.NewRMCRequest()
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.CallID = common_globals.CurrentMatchmakingCallID.Next()
	rmcRequest.Parameters = oEventBytes

	rmcRequestBytes := rmcRequest.Bytes()

	targetClient := server.FindClientByPID(uint64(ownerPID))
	if targetClient == nil {
		common_globals.Logger.Warning("Owner client not found")
		return 0
	}

	var messagePacket nex.PRUDPPacketInterface

	if server.PRUDPVersion == 0 {
		messagePacket, _ = nex.NewPRUDPPacketV0(targetClient, nil)
	} else {
		messagePacket, _ = nex.NewPRUDPPacketV1(targetClient, nil)
	}

	messagePacket.SetType(nex.DataPacket)
	messagePacket.AddFlag(nex.FlagNeedsAck)
	messagePacket.AddFlag(nex.FlagReliable)
	messagePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	messagePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	messagePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	messagePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	messagePacket.SetPayload(rmcRequestBytes)

	server.Send(messagePacket)

	return 0
}
