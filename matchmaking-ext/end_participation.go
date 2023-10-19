package match_making_ext

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func endParticipation(err error, client *nex.Client, callID uint32, idGathering uint32, strMessage string) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonMatchMakingExtProtocol.server
	var session *common_globals.CommonMatchmakeSession
	var ok bool
	if session, ok = common_globals.Sessions[idGathering]; !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	matchmakeSession := session.GameMatchmakeSession
	ownerPID := matchmakeSession.Gathering.OwnerPID

	var deleteSession bool = false
	if client.PID() == matchmakeSession.Gathering.OwnerPID {
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
		common_globals.RemoveConnectionIDFromSession(client.ConnectionID(), idGathering)
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteBool(true) // %retval%

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(match_making_ext.ProtocolID, callID)
	rmcResponse.SetSuccess(match_making_ext.MethodEndParticipation, rmcResponseBody)

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

	category := notifications.NotificationCategories.Participation
	subtype := notifications.NotificationSubTypes.Participation.Ended

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = client.PID()
	oEvent.Type = notifications.BuildNotificationType(category, subtype)
	oEvent.Param1 = idGathering
	oEvent.Param2 = client.PID()
	oEvent.StrParam = strMessage

	stream := nex.NewStreamOut(server)
	oEventBytes := oEvent.Bytes(stream)
	rmcMessage.SetParameters(oEventBytes)

	rmcMessageBytes := rmcMessage.Bytes()

	targetClient := server.FindClientFromPID(uint32(ownerPID))
	if targetClient == nil {
		logger.Warning("Owner client not found")
		return 0
	}

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
	messagePacket.SetPayload(rmcMessageBytes)

	messagePacket.AddFlag(nex.FlagNeedsAck)
	messagePacket.AddFlag(nex.FlagReliable)

	server.Send(messagePacket)

	return 0
}
