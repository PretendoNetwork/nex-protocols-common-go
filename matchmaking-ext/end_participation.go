package match_making_ext

import (
	"math"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
	"github.com/PretendoNetwork/nex-protocols-go/notifications"
)

func endParticipation(err error, client *nex.Client, callID uint32, idGathering uint32, strMessage string) {
	server := commonMatchMakingExtProtocol.server
	matchmakeSession := common_globals.Sessions[idGathering].GameMatchmakeSession
	ownerPID := matchmakeSession.Gathering.OwnerPID

	var deleteSession bool = false
	if client.PID() == matchmakeSession.Gathering.OwnerPID {
		// Flag 0x10 tells the server to change the matchmake session owner if they disconnect
		// If the flag is not set, delete the session
		// More info: https://nintendo-wiki.pretendo.network/docs/nex/protocols/match-making/types#flags
		if matchmakeSession.Gathering.Flags & 0x10 == 0 {
			deleteSession = true
		} else {
			changeSessionOwner(client.ConnectionID(), idGathering, callID)
		}
	}

	if deleteSession {
		delete(common_globals.Sessions, idGathering)
	} else {
		common_globals.RemoveConnectionIDFromRoom(client.ConnectionID(), idGathering)
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
	rmcMessage.SetCallID(0xffff0000 + callID)
	rmcMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

	oEvent := notifications.NewNotificationEvent()
	oEvent.PIDSource = client.PID()
	oEvent.Type = notifications.NotificationTypes.ParticipationEnded
	oEvent.Param1 = idGathering
	oEvent.Param2 = client.PID()
	oEvent.StrParam = strMessage

	stream := nex.NewStreamOut(server)
	oEventBytes := oEvent.Bytes(stream)
	rmcMessage.SetParameters(oEventBytes)

	rmcMessageBytes := rmcMessage.Bytes()

	targetClient := server.FindClientFromPID(uint32(ownerPID))

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
}

func changeSessionOwner(ownerConnectionID uint32, gathering uint32, callID uint32) {
	server := commonMatchMakingExtProtocol.server
	var otherClient *nex.Client

	otherConnectionID := common_globals.FindOtherConnectionID(ownerConnectionID, gathering)
	if otherConnectionID != math.MaxUint32 {
		otherClient = server.FindClientFromConnectionID(otherConnectionID)
		if otherClient != nil {
			common_globals.Sessions[gathering].GameMatchmakeSession.Gathering.OwnerPID = otherClient.PID()
		} else {
			logger.Warning("Other client not found")
			return
		}
	} else {
		return
	}

	rmcMessage := nex.NewRMCRequest()
	rmcMessage.SetProtocolID(notifications.ProtocolID)
	rmcMessage.SetCallID(0xffff0000 + callID)
	rmcMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

	oEvent := notifications.NewNotificationEvent()
	oEvent.PIDSource = otherClient.PID()
	oEvent.Type = notifications.NotificationTypes.OwnershipChanged
	oEvent.Param1 = gathering
	oEvent.Param2 = otherClient.PID()

	// TODO - This doesn't seem to appear on all servers
	// https://github.com/kinnay/NintendoClients/issues/101
	// unixTime := time.Now()
	// oEvent.StrParam = strconv.FormatInt(unixTime.UnixMicro(), 10)

	stream := nex.NewStreamOut(server)
	oEventBytes := oEvent.Bytes(stream)
	rmcMessage.SetParameters(oEventBytes)

	rmcRequestBytes := rmcMessage.Bytes()

	for _, connectionID := range common_globals.Sessions[gathering].ConnectionIDs {
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
}
