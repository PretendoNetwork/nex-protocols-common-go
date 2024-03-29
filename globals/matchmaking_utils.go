package common_globals

import (
	"crypto/rand"
	"fmt"
	"math"
	"strconv"
	"strings"

	nex "github.com/PretendoNetwork/nex-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	notifications "github.com/PretendoNetwork/nex-protocols-go/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/notifications/types"
	"golang.org/x/exp/slices"
)

// GetAvailableGatheringID returns a gathering ID which doesn't belong to any session
// Returns 0 if no IDs are available (math.MaxUint32 has been reached)
func GetAvailableGatheringID() uint32 {
	if CurrentGatheringID.Value() == math.MaxUint32 {
		return 0
	}

	return CurrentGatheringID.Increment()
}

// FindOtherConnectionID searches a connection ID on the session that isn't the given one
// Returns 0 if no connection ID could be found
func FindOtherConnectionID(excludedConnectionID uint32, gatheringID uint32) uint32 {
	for _, connectionID := range Sessions[gatheringID].ConnectionIDs {
		if connectionID != excludedConnectionID {
			return connectionID
		}
	}

	return 0
}

// RemoveConnectionIDFromSession removes a client from the session
func RemoveConnectionIDFromSession(clientConnectionID uint32, gathering uint32) {
	for index, connectionID := range Sessions[gathering].ConnectionIDs {
		if connectionID == clientConnectionID {
			Sessions[gathering].ConnectionIDs = DeleteIndex(Sessions[gathering].ConnectionIDs, index)
		}
	}

	if len(Sessions[gathering].ConnectionIDs) == 0 {
		delete(Sessions, gathering)
	}
}

// FindClientSession searches for session the given connection ID is connected to
func FindClientSession(connectionID uint32) uint32 {
	for gatheringID := range Sessions {
		if slices.Contains(Sessions[gatheringID].ConnectionIDs, connectionID) {
			return gatheringID
		}
	}

	return 0
}

// RemoveClientFromAllSessions removes a client from every session
func RemoveClientFromAllSessions(client *nex.Client) {
	// * Keep checking until no session is found
	for gid := FindClientSession(client.ConnectionID()); gid != 0; {
		session := Sessions[gid]
		lenParticipants := len(session.ConnectionIDs)

		RemoveConnectionIDFromSession(client.ConnectionID(), gid)

		if lenParticipants <= 1 {
			gid = FindClientSession(client.ConnectionID())
			continue
		}

		ownerPID := session.GameMatchmakeSession.Gathering.OwnerPID

		if client.PID() == ownerPID {
			// This flag tells the server to change the matchmake session owner if they disconnect
			// If the flag is not set, delete the session
			// More info: https://nintendo-wiki.pretendo.network/docs/nex/protocols/match-making/types#flags
			if session.GameMatchmakeSession.Gathering.Flags&match_making.GatheringFlags.DisconnectChangeOwner == 0 {
				delete(Sessions, gid)
			} else {
				ChangeSessionOwner(client, gid)
			}
		} else {
			server := client.Server()

			rmcMessage := nex.NewRMCRequest()
			rmcMessage.SetProtocolID(notifications.ProtocolID)
			rmcMessage.SetCallID(CurrentMatchmakingCallID.Increment())
			rmcMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

			category := notifications.NotificationCategories.Participation
			subtype := notifications.NotificationSubTypes.Participation.Disconnected

			oEvent := notifications_types.NewNotificationEvent()
			oEvent.PIDSource = client.PID()
			oEvent.Type = notifications.BuildNotificationType(category, subtype)
			oEvent.Param1 = gid
			oEvent.Param2 = client.PID()

			stream := nex.NewStreamOut(server)
			stream.WriteStructure(oEvent)
			rmcMessage.SetParameters(stream.Bytes())

			rmcMessageBytes := rmcMessage.Bytes()

			targetClient := server.FindClientFromPID(uint32(ownerPID))
			if targetClient == nil {
				Logger.Warning("Owner client not found")
				gid = FindClientSession(client.ConnectionID())
				continue
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
		}

		gid = FindClientSession(client.ConnectionID())
	}
}

// CreateSessionByMatchmakeSession creates a gathering from a MatchmakeSession
func CreateSessionByMatchmakeSession(matchmakeSession *match_making_types.MatchmakeSession, searchMatchmakeSession *match_making_types.MatchmakeSession, hostPID uint32) (*CommonMatchmakeSession, error, uint32) {
	sessionIndex := GetAvailableGatheringID()
	if sessionIndex == 0 {
		CurrentGatheringID = nex.NewCounter(0)
		sessionIndex = GetAvailableGatheringID()
	}

	session := CommonMatchmakeSession{
		SearchMatchmakeSession: searchMatchmakeSession,
		GameMatchmakeSession:   matchmakeSession,
	}

	session.GameMatchmakeSession.Gathering.ID = sessionIndex
	session.GameMatchmakeSession.Gathering.OwnerPID = hostPID
	session.GameMatchmakeSession.Gathering.HostPID = hostPID

	session.GameMatchmakeSession.StartedTime = nex.NewDateTime(0)
	session.GameMatchmakeSession.StartedTime.UTC()
	session.GameMatchmakeSession.SessionKey = make([]byte, 32)
	rand.Read(session.GameMatchmakeSession.SessionKey)

	if session.GameMatchmakeSession.MatchmakeParam == nil {
		session.GameMatchmakeSession.MatchmakeParam = match_making_types.NewMatchmakeParam()
	}

	if session.GameMatchmakeSession.MatchmakeParam.Parameters == nil {
		session.GameMatchmakeSession.MatchmakeParam.Parameters = make(map[string]*nex.Variant)
	}

	session.GameMatchmakeSession.MatchmakeParam.Parameters["@SR"] = nex.NewVariant()
	session.GameMatchmakeSession.MatchmakeParam.Parameters["@SR"].TypeID = 3
	session.GameMatchmakeSession.MatchmakeParam.Parameters["@SR"].Bool = true

	session.GameMatchmakeSession.MatchmakeParam.Parameters["@GIR"] = nex.NewVariant()
	session.GameMatchmakeSession.MatchmakeParam.Parameters["@GIR"].TypeID = 1
	session.GameMatchmakeSession.MatchmakeParam.Parameters["@GIR"].Int64 = 3

	Sessions[sessionIndex] = &session

	return Sessions[sessionIndex], nil, 0
}

// FindSessionByMatchmakeSession finds a gathering that matches with a MatchmakeSession
func FindSessionByMatchmakeSession(pid uint32, searchMatchmakeSession *match_making_types.MatchmakeSession) uint32 {
	// * This portion finds any sessions that match the search session
	// * It does not care about anything beyond that, such as if the match is already full
	// * This is handled below
	candidateSessionIndexes := make([]uint32, 0, len(Sessions))
	for index, session := range Sessions {
		if session.SearchMatchmakeSession.Equals(searchMatchmakeSession) {
			candidateSessionIndexes = append(candidateSessionIndexes, index)
		}
	}

	var friendList []uint32
	for _, sessionIndex := range candidateSessionIndexes {
		sessionToCheck := Sessions[sessionIndex]
		if len(sessionToCheck.ConnectionIDs) >= int(sessionToCheck.GameMatchmakeSession.MaximumParticipants) {
			continue
		}

		if !sessionToCheck.GameMatchmakeSession.OpenParticipation {
			continue
		}

		// If the session only allows friends, check if the owner is in the friend list of the PID
		// TODO - Is this a flag or a constant?
		if sessionToCheck.GameMatchmakeSession.ParticipationPolicy == 98 {
			if GetUserFriendPIDsHandler == nil {
				Logger.Warning("Missing GetUserFriendPIDsHandler!")
				continue
			}

			if len(friendList) == 0 {
				friendList = GetUserFriendPIDsHandler(pid)
			}

			if !slices.Contains(friendList, sessionToCheck.GameMatchmakeSession.OwnerPID) {
				continue
			}
		}

		return sessionIndex // * Found a match
	}

	return 0
}

// FindSessionsByMatchmakeSessionSearchCriterias finds a gathering that matches with the given search criteria
func FindSessionsByMatchmakeSessionSearchCriterias(pid uint32, lstSearchCriteria []*match_making_types.MatchmakeSessionSearchCriteria, gameSpecificChecks func(searchCriteria *match_making_types.MatchmakeSessionSearchCriteria, matchmakeSession *match_making_types.MatchmakeSession) bool) []*CommonMatchmakeSession {
	candidateSessions := make([]*CommonMatchmakeSession, 0, len(Sessions))
	var friendList []uint32
	for _, session := range Sessions {
		for _, searchCriteria := range lstSearchCriteria {
			// * Check things like game specific attributes
			if gameSpecificChecks != nil {
				if !gameSpecificChecks(searchCriteria, session.GameMatchmakeSession) {
					continue
				}
			} else {
				if !compareAttributesSearchCriteria(session.GameMatchmakeSession.Attributes, searchCriteria.Attribs) {
					continue
				}
			}

			if !compareSearchCriteria(session.GameMatchmakeSession.MaximumParticipants, searchCriteria.MaxParticipants) {
				continue
			}

			if !compareSearchCriteria(session.GameMatchmakeSession.MinimumParticipants, searchCriteria.MinParticipants) {
				continue
			}

			if !compareSearchCriteria(session.GameMatchmakeSession.GameMode, searchCriteria.GameMode) {
				continue
			}

			if len(session.ConnectionIDs) >= int(session.GameMatchmakeSession.MaximumParticipants) {
				continue
			}

			if !session.GameMatchmakeSession.OpenParticipation {
				continue
			}

			// If the session only allows friends, check if the owner is in the friend list of the PID
			// TODO - Is this a flag or a constant?
			if session.GameMatchmakeSession.ParticipationPolicy == 98 {
				if GetUserFriendPIDsHandler == nil {
					Logger.Warning("Missing GetUserFriendPIDsHandler!")
					continue
				}

				if len(friendList) == 0 {
					friendList = GetUserFriendPIDsHandler(pid)
				}

				if !slices.Contains(friendList, session.GameMatchmakeSession.OwnerPID) {
					continue
				}
			}

			candidateSessions = append(candidateSessions, session)

			// We don't have to compare with other search criterias
			break
		}
	}

	return candidateSessions
}

func compareAttributesSearchCriteria(original []uint32, search []string) bool {
	if len(original) != len(search) {
		return false
	}

	for index, originalAttribute := range original {
		searchAttribute := search[index]

		if !compareSearchCriteria(originalAttribute, searchAttribute) {
			return false
		}
	}

	return true
}

func compareSearchCriteria[T ~uint16 | ~uint32](original T, search string) bool {
	if search == "" { // Accept any value
		return true
	}

	before, after, found := strings.Cut(search, ",")
	if found {
		min, err := strconv.ParseUint(before, 10, 64)
		if err != nil {
			return false
		}

		max, err := strconv.ParseUint(after, 10, 64)
		if err != nil {
			return false
		}

		return min <= uint64(original) && max >= uint64(original)
	} else {
		searchNum, err := strconv.ParseUint(before, 10, 64)
		if err != nil {
			return false
		}

		return searchNum == uint64(original)
	}
}

// AddPlayersToSession updates the given sessions state to include the provided connection IDs
// Returns a NEX error code if failed
func AddPlayersToSession(session *CommonMatchmakeSession, connectionIDs []uint32, initiatingClient *nex.Client, joinMessage string) (error, uint32) {
	if (len(session.ConnectionIDs) + len(connectionIDs)) > int(session.GameMatchmakeSession.Gathering.MaximumParticipants) {
		return fmt.Errorf("Gathering %d is full", session.GameMatchmakeSession.Gathering.ID), nex.Errors.RendezVous.SessionFull
	}

	for _, connectedID := range connectionIDs {
		if slices.Contains(session.ConnectionIDs, connectedID) {
			return fmt.Errorf("Connection ID %d is already in gathering %d", connectedID, session.GameMatchmakeSession.Gathering.ID), nex.Errors.RendezVous.AlreadyParticipatedGathering
		}

		session.ConnectionIDs = append(session.ConnectionIDs, connectedID)

		session.GameMatchmakeSession.ParticipationCount += 1
	}

	server := initiatingClient.Server()

	for i := 0; i < len(session.ConnectionIDs); i++ {
		target := server.FindClientFromConnectionID(session.ConnectionIDs[i])
		if target == nil {
			// TODO - Error here?
			Logger.Warning("Player not found")
			continue
		}

		notificationRequestMessage := nex.NewRMCRequest()
		notificationRequestMessage.SetProtocolID(notifications.ProtocolID)
		notificationRequestMessage.SetCallID(CurrentMatchmakingCallID.Increment())
		notificationRequestMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

		notificationCategory := notifications.NotificationCategories.Participation
		notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = initiatingClient.PID()
		oEvent.Type = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
		oEvent.Param1 = session.GameMatchmakeSession.ID
		oEvent.Param2 = target.PID()
		oEvent.StrParam = joinMessage
		oEvent.Param3 = uint32(len(connectionIDs))

		notificationStream := nex.NewStreamOut(server)

		notificationStream.WriteStructure(oEvent)

		notificationRequestMessage.SetParameters(notificationStream.Bytes())
		notificationRequestBytes := notificationRequestMessage.Bytes()

		var messagePacket nex.PacketInterface

		if server.PRUDPVersion() == 0 {
			messagePacket, _ = nex.NewPacketV0(target, nil)
			messagePacket.SetVersion(0)
		} else {
			messagePacket, _ = nex.NewPacketV1(target, nil)
			messagePacket.SetVersion(1)
		}

		messagePacket.SetSource(0xA1)
		messagePacket.SetDestination(0xAF)
		messagePacket.SetType(nex.DataPacket)
		messagePacket.SetPayload(notificationRequestBytes)

		messagePacket.AddFlag(nex.FlagNeedsAck)
		messagePacket.AddFlag(nex.FlagReliable)

		server.Send(messagePacket)
	}

	// This appears to be correct. Tri-Force Heroes uses 3.9.0, and has issues if these notifications are sent
	// Minecraft, however, requires these to be sent
	// TODO: Check other games both pre and post 3.10.0 and validate
	if server.MatchMakingProtocolVersion().GreaterOrEqual("3.10.0") {
		for i := 0; i < len(session.ConnectionIDs); i++ {
			target := server.FindClientFromConnectionID(session.ConnectionIDs[i])
			if target == nil {
				// TODO - Error here?
				Logger.Warning("Player not found")
				continue
			}

			notificationRequestMessage := nex.NewRMCRequest()
			notificationRequestMessage.SetProtocolID(notifications.ProtocolID)
			notificationRequestMessage.SetCallID(CurrentMatchmakingCallID.Increment())
			notificationRequestMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

			notificationCategory := notifications.NotificationCategories.Participation
			notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

			oEvent := notifications_types.NewNotificationEvent()
			oEvent.PIDSource = initiatingClient.PID()
			oEvent.Type = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
			oEvent.Param1 = session.GameMatchmakeSession.ID
			oEvent.Param2 = target.PID()
			oEvent.StrParam = joinMessage
			oEvent.Param3 = uint32(len(connectionIDs))

			notificationStream := nex.NewStreamOut(server)

			notificationStream.WriteStructure(oEvent)

			notificationRequestMessage.SetParameters(notificationStream.Bytes())
			notificationRequestBytes := notificationRequestMessage.Bytes()

			var messagePacket nex.PacketInterface

			if server.PRUDPVersion() == 0 {
				messagePacket, _ = nex.NewPacketV0(initiatingClient, nil)
				messagePacket.SetVersion(0)
			} else {
				messagePacket, _ = nex.NewPacketV1(initiatingClient, nil)
				messagePacket.SetVersion(1)
			}

			messagePacket.SetSource(0xA1)
			messagePacket.SetDestination(0xAF)
			messagePacket.SetType(nex.DataPacket)
			messagePacket.SetPayload(notificationRequestBytes)

			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)

			server.Send(messagePacket)
		}

		notificationRequestMessage := nex.NewRMCRequest()
		notificationRequestMessage.SetProtocolID(notifications.ProtocolID)
		notificationRequestMessage.SetCallID(CurrentMatchmakingCallID.Increment())
		notificationRequestMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

		notificationCategory := notifications.NotificationCategories.Participation
		notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = initiatingClient.PID()
		oEvent.Type = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
		oEvent.Param1 = session.GameMatchmakeSession.ID
		oEvent.Param2 = initiatingClient.PID()
		oEvent.StrParam = joinMessage
		oEvent.Param3 = uint32(len(connectionIDs))

		notificationStream := nex.NewStreamOut(server)

		notificationStream.WriteStructure(oEvent)

		notificationRequestMessage.SetParameters(notificationStream.Bytes())
		notificationRequestBytes := notificationRequestMessage.Bytes()

		var messagePacket nex.PacketInterface

		if server.PRUDPVersion() == 0 {
			messagePacket, _ = nex.NewPacketV0(initiatingClient, nil)
			messagePacket.SetVersion(0)
		} else {
			messagePacket, _ = nex.NewPacketV1(initiatingClient, nil)
			messagePacket.SetVersion(1)
		}

		messagePacket.SetSource(0xA1)
		messagePacket.SetDestination(0xAF)
		messagePacket.SetType(nex.DataPacket)
		messagePacket.SetPayload(notificationRequestBytes)

		messagePacket.AddFlag(nex.FlagNeedsAck)
		messagePacket.AddFlag(nex.FlagReliable)

		server.Send(messagePacket)

		target := server.FindClientFromPID(uint32(session.GameMatchmakeSession.Gathering.OwnerPID))
		if target == nil {
			// TODO - Error here?
			Logger.Warning("Player not found")
			return nil, 0
		}

		if server.PRUDPVersion() == 0 {
			messagePacket, _ = nex.NewPacketV0(target, nil)
			messagePacket.SetVersion(0)
		} else {
			messagePacket, _ = nex.NewPacketV1(target, nil)
			messagePacket.SetVersion(1)
		}

		messagePacket.SetSource(0xA1)
		messagePacket.SetDestination(0xAF)
		messagePacket.SetType(nex.DataPacket)
		messagePacket.SetPayload(notificationRequestBytes)

		messagePacket.AddFlag(nex.FlagNeedsAck)
		messagePacket.AddFlag(nex.FlagReliable)

		server.Send(messagePacket)
	}

	return nil, 0
}

// ChangeSessionOwner changes the session owner to a different client
func ChangeSessionOwner(ownerClient *nex.Client, gathering uint32) {
	server := ownerClient.Server()
	var otherClient *nex.Client

	otherConnectionID := FindOtherConnectionID(ownerClient.ConnectionID(), gathering)
	if otherConnectionID != 0 {
		otherClient = server.FindClientFromConnectionID(uint32(otherConnectionID))
		if otherClient != nil {
			Sessions[gathering].GameMatchmakeSession.Gathering.OwnerPID = otherClient.PID()
		} else {
			Logger.Warning("Other client not found")
			return
		}
	} else {
		return
	}

	rmcMessage := nex.NewRMCRequest()
	rmcMessage.SetProtocolID(notifications.ProtocolID)
	rmcMessage.SetCallID(CurrentMatchmakingCallID.Increment())
	rmcMessage.SetMethodID(notifications.MethodProcessNotificationEvent)

	category := notifications.NotificationCategories.OwnershipChanged
	subtype := notifications.NotificationSubTypes.OwnershipChanged.None

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = otherClient.PID()
	oEvent.Type = notifications.BuildNotificationType(category, subtype)
	oEvent.Param1 = gathering
	oEvent.Param2 = otherClient.PID()

	// TODO - StrParam doesn't have this value on some servers
	// https://github.com/kinnay/NintendoClients/issues/101
	// unixTime := time.Now()
	// oEvent.StrParam = strconv.FormatInt(unixTime.UnixMicro(), 10)

	stream := nex.NewStreamOut(server)
	stream.WriteStructure(oEvent)
	rmcMessage.SetParameters(stream.Bytes())

	rmcRequestBytes := rmcMessage.Bytes()

	for _, connectionID := range Sessions[gathering].ConnectionIDs {
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
			Logger.Warning("Client not found")
		}
	}
}
