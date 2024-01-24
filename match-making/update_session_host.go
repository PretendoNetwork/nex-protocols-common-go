package matchmaking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func updateSessionHost(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32, isMigrateOwner *types.PrimitiveBool) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodesCore.InvalidArgument
	}

	session, ok := common_globals.Sessions[gid.Value]
	if !ok {
		return nil, nex.ResultCodesRendezVous.SessionVoid
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	if common_globals.FindConnectionSession(connection.ID) != gid.Value {
		return nil, nex.ResultCodesRendezVous.PermissionDenied
	}

	session.GameMatchmakeSession.Gathering.HostPID = connection.PID().Copy().(*types.PID)

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodUpdateSessionHost
	rmcResponse.CallID = callID

	if !isMigrateOwner.Value {
		return rmcResponse, 0
	}

	session.GameMatchmakeSession.Gathering.OwnerPID = connection.PID().Copy().(*types.PID)

	category := notifications.NotificationCategories.OwnershipChanged
	subtype := notifications.NotificationSubTypes.OwnershipChanged.None

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID().Copy().(*types.PID)
	oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(category, subtype))
	oEvent.Param1 = gid.Copy().(*types.PrimitiveU32)
	oEvent.Param2 = types.NewPrimitiveU32(connection.PID().LegacyValue()) // TODO - This assumes a legacy client. Will not work on the Switch

	// TODO - StrParam doesn't have this value on some servers
	// * https://github.com/kinnay/NintendoClients/issues/101
	// * unixTime := time.Now()
	// * oEvent.StrParam = strconv.FormatInt(unixTime.UnixMicro(), 10)

	stream := nex.NewByteStreamOut(server)

	oEvent.WriteTo(stream)

	rmcRequest := nex.NewRMCRequest(server)
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.CallID = common_globals.CurrentMatchmakingCallID.Next()
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.Parameters = stream.Bytes()

	rmcRequestBytes := rmcRequest.Bytes()

	for _, connectionID := range common_globals.Sessions[gid.Value].ConnectionIDs {
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
