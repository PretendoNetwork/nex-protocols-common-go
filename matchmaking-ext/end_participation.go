package match_making_ext

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	"github.com/PretendoNetwork/nex-protocols-go/notifications"
)

func EndParticipation(err error, client *nex.Client, callID uint32, idGathering uint32, strMessage string) {
	server := commonMatchMakingExtProtocol.server
	matchmakeSession := common_globals.Sessions[idGathering].GameMatchmakeSession
	ownerPID := matchmakeSession.Gathering.OwnerPID
	if client.PID() == matchmakeSession.Gathering.OwnerPID {
		if matchmakeSession.Gathering.Flags & 0x10 == 0 {
			delete(common_globals.Sessions, idGathering)
		}
	}

	common_globals.RemoveConnectionIDFromRoom(client.ConnectionID(), idGathering)

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
