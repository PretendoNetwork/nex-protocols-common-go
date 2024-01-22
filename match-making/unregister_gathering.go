package matchmaking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func unregisterGathering(err error, packet nex.PacketInterface, callID uint32, idGathering *types.PrimitiveU32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[idGathering.Value]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	if session.GameMatchmakeSession.Gathering.OwnerPID.Equals(connection.PID()) {
		return nil, nex.Errors.RendezVous.PermissionDenied
	}

	gatheringPlayers := session.ConnectionIDs

	delete(common_globals.Sessions, idGathering.Value)

	retval := types.NewPrimitiveBool(true)

	rmcResponseStream := nex.NewByteStreamOut(server)

	retval.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodUnregisterGathering
	rmcResponse.CallID = callID

	category := notifications.NotificationCategories.GatheringUnregistered
	subtype := notifications.NotificationSubTypes.GatheringUnregistered.None

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID().Copy().(*types.PID)
	oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(category, subtype))
	oEvent.Param1 = idGathering.Copy().(*types.PrimitiveU32)

	stream := nex.NewByteStreamOut(server)

	oEvent.WriteTo(stream)

	rmcRequest := nex.NewRMCRequest(server)
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.CallID = common_globals.CurrentMatchmakingCallID.Next()
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.Parameters = stream.Bytes()

	rmcRequestBytes := rmcRequest.Bytes()

	for _, connectionID := range gatheringPlayers {
		target := endpoint.FindConnectionByID(connectionID)
		if target == nil {
			common_globals.Logger.Warning("Client not found")
			continue
		}

		var messagePacket nex.PRUDPPacketInterface

		if target.DefaultPRUDPVersion == 0 {
			messagePacket, _ = nex.NewPRUDPPacketV0(target, nil)
		} else {
			messagePacket, _ = nex.NewPRUDPPacketV1(target, nil)
		}

		messagePacket.SetType(nex.DataPacket)
		messagePacket.AddFlag(nex.FlagNeedsAck)
		messagePacket.AddFlag(nex.FlagReliable)
		messagePacket.SetSourceVirtualPortStreamType(target.StreamType)
		messagePacket.SetSourceVirtualPortStreamID(target.Endpoint.StreamID)
		messagePacket.SetDestinationVirtualPortStreamType(target.StreamType)
		messagePacket.SetDestinationVirtualPortStreamID(target.StreamID)
		messagePacket.SetPayload(rmcRequestBytes)

		server.Send(messagePacket)
	}

	return rmcResponse, 0
}