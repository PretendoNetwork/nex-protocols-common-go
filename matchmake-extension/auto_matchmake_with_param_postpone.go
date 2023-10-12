package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func autoMatchmakeWithParam_Postpone(err error, client *nex.Client, callID uint32, autoMatchmakeParam *match_making_types.AutoMatchmakeParam) uint32 {
	if commonMatchmakeExtensionProtocol.cleanupSearchMatchmakeSessionHandler == nil {
		logger.Warning("MatchmakeExtension::AutoMatchmake_Postpone missing CleanupSearchMatchmakeSessionHandler!")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonMatchmakeExtensionProtocol.server

	// A client may disconnect from a session without leaving reliably,
	// so let's make sure the client is removed from the session
	common_globals.RemoveClientFromAllSessions(client)

	var matchmakeSession *match_making_types.MatchmakeSession
	matchmakeSession = autoMatchmakeParam.SourceMatchmakeSession

	searchMatchmakeSession := matchmakeSession.Copy().(*match_making_types.MatchmakeSession)
	commonMatchmakeExtensionProtocol.cleanupSearchMatchmakeSessionHandler(searchMatchmakeSession)
	sessionIndex := common_globals.FindSessionByMatchmakeSession(searchMatchmakeSession)
	var session *common_globals.CommonMatchmakeSession

	if sessionIndex == 0 {
		var errCode uint32
		session, err, errCode = common_globals.CreateSessionByMatchmakeSession(matchmakeSession, searchMatchmakeSession, client.PID())
		if err != nil {
			logger.Error(err.Error())
			return errCode
		}
	}else{
		session = common_globals.Sessions[sessionIndex]
	}
		/*sessionIndex = common_globals.GetAvailableGatheringID()
		// This should in theory be impossible, as there aren't enough PIDs creating sessions to fill the uint32 limit.
		// If we ever get here, we must be not deleting sessions properly
		if sessionIndex == 0 {
			logger.Critical("No gatherings available!")
			return nex.Errors.RendezVous.LimitExceeded
		}

		session := common_globals.CommonMatchmakeSession{
			SearchMatchmakeSession: searchMatchmakeSession,
			GameMatchmakeSession:   matchmakeSession,
		}

		session = &session
		session.GameMatchmakeSession.Gathering.ID = sessionIndex
		session.GameMatchmakeSession.Gathering.OwnerPID = client.PID()
		session.GameMatchmakeSession.Gathering.HostPID = client.PID()

		session.GameMatchmakeSession.StartedTime = nex.NewDateTime(0)
		session.GameMatchmakeSession.StartedTime.UTC()
		session.GameMatchmakeSession.SessionKey = make([]byte, 32)

		session.GameMatchmakeSession.MatchmakeParam.Parameters["@SR"] = nex.NewVariant()
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@SR"].TypeID = 3
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@SR"].Bool = true
	
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@GIR"] = nex.NewVariant()
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@GIR"].TypeID = 1
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@GIR"].Int64 = 3
	
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@Lon"] = nex.NewVariant()
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@Lon"].TypeID = 2
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@Lon"].Float64 = 0
	
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@Lat"] = nex.NewVariant()
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@Lat"].TypeID = 2
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@Lat"].Float64 = 0
	
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@CC"] = nex.NewVariant()
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@CC"].TypeID = 4
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@CC"].Str = "US"
	
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@DR"] = nex.NewVariant()
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@DR"].TypeID = 1
		session.GameMatchmakeSession.MatchmakeParam.Parameters["@DR"].Int64 = 0
	}*/

	err, errCode := common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID()}, client)
	if err != nil {
		logger.Error(err.Error())
		return errCode
	}

	rmcResponseStream := nex.NewStreamOut(server)
	matchmakeDataHolder := nex.NewDataHolder()
	matchmakeDataHolder.SetTypeName("MatchmakeSession")
	matchmakeDataHolder.SetObjectData(session.GameMatchmakeSession)
	rmcResponseStream.WriteStructure(session.GameMatchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodAutoMatchmakeWithParamPostpone, rmcResponseBody)

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

	return 0
}
