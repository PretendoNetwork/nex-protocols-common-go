package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
)

func autoMatchmakeWithSearchCriteria_Postpone(err error, client *nex.Client, callID uint32, lstSearchCriteria []*match_making_types.MatchmakeSessionSearchCriteria, anyGathering *nex.DataHolder, message string) uint32 {
	if commonMatchmakeExtensionProtocol.cleanupMatchmakeSessionSearchCriteriaHandler == nil {
		logger.Warning("MatchmakeExtension::AutoMatchmakeWithSearchCriteria_Postpone missing CleanupMatchmakeSessionSearchCriteriaHandler!")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	commonMatchmakeExtensionProtocol.cleanupMatchmakeSessionSearchCriteriaHandler(lstSearchCriteria)

	server := commonMatchmakeExtensionProtocol.server

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	common_globals.RemoveClientFromAllSessions(client)

	var matchmakeSession *match_making_types.MatchmakeSession
	anyGatheringDataType := anyGathering.TypeName()

	if anyGatheringDataType == "MatchmakeSession" {
		matchmakeSession = anyGathering.ObjectData().(*match_making_types.MatchmakeSession)
	} else {
		logger.Critical("Non-MatchmakeSession DataType?!")
		return nex.Errors.Core.InvalidArgument
	}

	sessions := common_globals.FindSessionsByMatchmakeSessionSearchCriterias(lstSearchCriteria, commonMatchmakeExtensionProtocol.gameSpecificMatchmakeSessionSearchCriteriaChecksHandler)
	var session *common_globals.CommonMatchmakeSession

	if len(sessions) == 0 {
		gatheringID := common_globals.GetAvailableGatheringID()

		// * This should in theory be impossible, as there aren't enough PIDs creating sessions to fill the uint32 limit.
		// * If we ever get here, we must be not deleting sessions properly
		if gatheringID == 0 {
			logger.Critical("No gatherings available!")
			return nex.Errors.RendezVous.LimitExceeded
		}

		session = &common_globals.CommonMatchmakeSession{
			SearchCriteria:       lstSearchCriteria,
			GameMatchmakeSession: matchmakeSession,
		}

		session.GameMatchmakeSession.Gathering.ID = gatheringID
		session.GameMatchmakeSession.Gathering.OwnerPID = client.PID()
		session.GameMatchmakeSession.Gathering.HostPID = client.PID()
		session.GameMatchmakeSession.StartedTime = nex.NewDateTime(0)
		session.GameMatchmakeSession.StartedTime.UTC()
		session.GameMatchmakeSession.SessionKey = make([]byte, 32)

		common_globals.Sessions[gatheringID] = session
	} else {
		session = sessions[0]
	}

	session.ConnectionIDs = append(session.ConnectionIDs, client.ConnectionID())
	session.GameMatchmakeSession.ParticipationCount = uint32(len(session.ConnectionIDs))

	rmcResponseStream := nex.NewStreamOut(server)
	matchmakeDataHolder := nex.NewDataHolder()
	matchmakeDataHolder.SetTypeName("MatchmakeSession")
	matchmakeDataHolder.SetObjectData(session.GameMatchmakeSession)
	rmcResponseStream.WriteDataHolder(matchmakeDataHolder)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodAutoMatchmakeWithSearchCriteriaPostpone, rmcResponseBody)

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

	category := notifications.NotificationCategories.Participation
	subtype := notifications.NotificationSubTypes.Participation.NewParticipant

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = session.GameMatchmakeSession.Gathering.HostPID
	oEvent.Type = notifications.BuildNotificationType(category, subtype)
	oEvent.Param1 = session.GameMatchmakeSession.Gathering.ID
	oEvent.Param2 = client.PID()
	oEvent.StrParam = message

	stream := nex.NewStreamOut(server)
	oEventBytes := oEvent.Bytes(stream)
	rmcMessage.SetParameters(oEventBytes)
	rmcMessageBytes := rmcMessage.Bytes()

	targetClient := server.FindClientFromPID(uint32(session.GameMatchmakeSession.Gathering.OwnerPID))
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
