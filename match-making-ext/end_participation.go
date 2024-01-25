package match_making_ext

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func endParticipation(err error, packet nex.PacketInterface, callID uint32, idGathering *types.PrimitiveU32, strMessage *types.String) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[idGathering.Value]
	if !ok {
		return nil, nex.ResultCodes.RendezVous.SessionVoid
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	matchmakeSession := session.GameMatchmakeSession
	ownerPID := matchmakeSession.Gathering.OwnerPID

	var deleteSession bool = false
	if connection.PID().Equals(matchmakeSession.Gathering.OwnerPID) {
		// * This flag tells the server to change the matchmake session owner if they disconnect
		// * If the flag is not set, delete the session
		// * More info: https://nintendo-wiki.pretendo.network/docs/nex/protocols/match-making/types#flags
		if matchmakeSession.Gathering.Flags.PAND(match_making.GatheringFlags.DisconnectChangeOwner) == 0 {
			deleteSession = true
		} else {
			common_globals.ChangeSessionOwner(connection, idGathering.Value)
		}
	}

	if deleteSession {
		delete(common_globals.Sessions, idGathering.Value)
	} else {
		common_globals.RemoveConnectionIDFromSession(connection.ID, idGathering.Value)
	}

	retval := types.NewPrimitiveBool(true)

	rmcResponseStream := nex.NewByteStreamOut(server)

	retval.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = match_making_ext.ProtocolID
	rmcResponse.MethodID = match_making_ext.MethodEndParticipation
	rmcResponse.CallID = callID

	category := notifications.NotificationCategories.Participation
	subtype := notifications.NotificationSubTypes.Participation.Ended

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID().Copy().(*types.PID)
	oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(category, subtype))
	oEvent.Param1 = idGathering.Copy().(*types.PrimitiveU32)
	oEvent.Param2 = types.NewPrimitiveU32(connection.PID().LegacyValue()) // TODO - This assumes a legacy client. Will not work on the Switch
	oEvent.StrParam = strMessage.Copy().(*types.String)

	stream := nex.NewByteStreamOut(server)

	oEvent.WriteTo(stream)

	rmcRequest := nex.NewRMCRequest(server)
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.CallID = common_globals.CurrentMatchmakingCallID.Next()
	rmcRequest.Parameters = stream.Bytes()

	rmcRequestBytes := rmcRequest.Bytes()

	target := endpoint.FindConnectionByPID(ownerPID.Value())
	if target == nil {
		common_globals.Logger.Warning("Owner client not found")
		return nil, 0
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

	return rmcResponse, 0
}
