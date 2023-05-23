package matchmake_extension

import (
	"math"

	nex "github.com/PretendoNetwork/nex-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	"github.com/PretendoNetwork/nex-protocols-go/notifications"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	"encoding/json"
)

func AutoMatchmake_Postpone(err error, client *nex.Client, callID uint32, matchmakeSession *match_making.MatchmakeSession, message string) {
	server := commonMatchmakeExtensionProtocol.server
	if commonMatchmakeExtensionProtocol.CleanupSearchMatchmakeSessionHandler == nil {
		logger.Warning("MatchmakeExtension::AutoMatchmake_Postpone missing CleanupSearchMatchmakeSessionHandler!")
		return
	}
	//This is the best way I found to copy the full object. And I still hate it.
	// TODO - for Jon: Copy in MatchmakeSession (and anything else that needs it)
	tmp, _ := json.Marshal(matchmakeSession)
	matchmakeSessionCopy := match_making.NewMatchmakeSession()
	json.Unmarshal(tmp, &matchmakeSessionCopy)
	searchMatchmakeSession := commonMatchmakeExtensionProtocol.CleanupSearchMatchmakeSessionHandler(*matchmakeSessionCopy)
	sessionIndex := uint32(common_globals.FindSearchMatchmakeSession(searchMatchmakeSession))
	if sessionIndex == math.MaxUint32 {
		session := common_globals.CommonMatchmakeSession{
			SearchMatchmakeSession: searchMatchmakeSession,
			GameMatchmakeSession:   *matchmakeSession,
		}
		sessionIndex = common_globals.CurrentGatheringID
		common_globals.Sessions[sessionIndex] = &session
		common_globals.Sessions[sessionIndex].GameMatchmakeSession.Gathering.ID = uint32(sessionIndex)
		common_globals.Sessions[sessionIndex].GameMatchmakeSession.Gathering.OwnerPID = client.PID()
		common_globals.Sessions[sessionIndex].GameMatchmakeSession.Gathering.HostPID = client.PID()
		common_globals.CurrentGatheringID++
	}

	common_globals.Sessions[sessionIndex].ConnectionIDs = append(common_globals.Sessions[sessionIndex].ConnectionIDs, client.ConnectionID())

	rmcResponseStream := nex.NewStreamOut(server)
	matchmakeDataHolder := nex.NewDataHolder()
	matchmakeDataHolder.SetTypeName("MatchmakeSession")
	matchmakeDataHolder.SetObjectData(&common_globals.Sessions[sessionIndex].GameMatchmakeSession)
	rmcResponseStream.WriteDataHolder(matchmakeDataHolder)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodAutoMatchmake_Postpone, rmcResponseBody)

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
	oEvent.PIDSource = common_globals.Sessions[sessionIndex].GameMatchmakeSession.Gathering.HostPID
	// TODO - for Jon: notifications type to make this simpler
	oEvent.Type = 3001 // New participant
	oEvent.Param1 = uint32(sessionIndex)
	oEvent.Param2 = client.PID()

	stream := nex.NewStreamOut(server)
	oEventBytes := oEvent.Bytes(stream)
	rmcMessage.SetParameters(oEventBytes)
	rmcMessageBytes := rmcMessage.Bytes()

	targetClient := server.FindClientFromPID(uint32(common_globals.Sessions[sessionIndex].GameMatchmakeSession.Gathering.HostPID))

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
