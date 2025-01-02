package match_making_ext

import (
	"fmt"
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/v2/match-making-ext"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

func (commonProtocol *CommonProtocol) endParticipation(err error, packet nex.PacketInterface, callID uint32, idGathering *types.PrimitiveU32, strMessage *types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	session, ok := common_globals.Sessions[idGathering.Value]
	if !ok {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, fmt.Sprintf("EndParticipation: PID %v requested to leave gathering %v, but it doesn't exist", packet.Sender().PID().Value(), idGathering.Value))
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	matchmakeSession := session.GameMatchmakeSession

	var deleteSession bool = false
	if connection.PID().Equals(matchmakeSession.Gathering.OwnerPID) {
		// * This flag tells the server to change the matchmake session owner if they disconnect
		// * If the flag is not set, delete the session
		// * More info: https://nintendo-wiki.pretendo.network/docs/nex/protocols/match-making/types#flags
		if matchmakeSession.Gathering.Flags.PAND(match_making.GatheringFlags.DisconnectChangeOwner) == 0 {
			deleteSession = true
		} else {
			common_globals.Logger.Infof("Owner PID %v leaving - attempting owner migration for gathering %v", connection.PID().Value(), idGathering.Value)
			common_globals.ChangeSessionOwner(connection, idGathering.Value, true)
		}
	}

	if deleteSession {
		// Shouldn't this.. notify the other clients...?
		common_globals.Logger.Infof("Owner PID %v left - deleting gathering %v", connection.PID().Value(), idGathering.Value)
		delete(common_globals.Sessions, idGathering.Value)
	} else {
		common_globals.Logger.Infof("PID %v left gathering %v", connection.PID().Value(), idGathering.Value)
		common_globals.RemoveConnectionIDFromSession(connection.ID, idGathering.Value)
	}

	retval := types.NewPrimitiveBool(true)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
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

	stream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	oEvent.WriteTo(stream)

	rmcRequest := nex.NewRMCRequest(endpoint)
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.CallID = common_globals.CurrentMatchmakingCallID.Next()
	rmcRequest.Parameters = stream.Bytes()

	rmcRequestBytes := rmcRequest.Bytes()

	target := endpoint.FindConnectionByID(session.OwnerConnectionID)
	if target == nil {
		common_globals.Logger.Warning("Owner client not found")
		return nil, nil
	}

	var messagePacket nex.PRUDPPacketInterface

	if target.DefaultPRUDPVersion == 0 {
		messagePacket, _ = nex.NewPRUDPPacketV0(server, target, nil)
	} else {
		messagePacket, _ = nex.NewPRUDPPacketV1(server, target, nil)
	}

	messagePacket.SetType(constants.DataPacket)
	messagePacket.AddFlag(constants.PacketFlagNeedsAck)
	messagePacket.AddFlag(constants.PacketFlagReliable)
	messagePacket.SetSourceVirtualPortStreamType(target.StreamType)
	messagePacket.SetSourceVirtualPortStreamID(endpoint.StreamID)
	messagePacket.SetDestinationVirtualPortStreamType(target.StreamType)
	messagePacket.SetDestinationVirtualPortStreamID(target.StreamID)
	messagePacket.SetPayload(rmcRequestBytes)

	server.Send(messagePacket)

	if commonProtocol.OnAfterEndParticipation != nil {
		go commonProtocol.OnAfterEndParticipation(packet, idGathering, strMessage)
	}

	return rmcResponse, nil
}
