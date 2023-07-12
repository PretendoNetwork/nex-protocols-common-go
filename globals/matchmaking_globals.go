package common_globals

import (
	"math"

	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
)

type CommonMatchmakeSession struct {
	GameMatchmakeSession   *match_making_types.MatchmakeSession // Used by the game, contains the current state of the MatchmakeSession
	SearchMatchmakeSession *match_making_types.MatchmakeSession // Used by the server when searching for matches, contains the state of the MatchmakeSession during the search process for easy compares
	SearchCriteria		   []*match_making_types.MatchmakeSessionSearchCriteria // Used by the server when searching for matches, contains the list of MatchmakeSessionSearchCriteria
	ConnectionIDs          []uint32                             // Players in the room, referenced by their connection IDs. This is used instead of the PID in order to ensure we're talking to the correct client (in case of e.g. multiple logins)
}

var Sessions map[uint32]*CommonMatchmakeSession

// GetSessionIndex returns a gathering ID which doesn't belong to any session
func GetSessionIndex() uint32 {
	var gatheringID uint32 = 1
	for gatheringID < math.MaxUint32 {
		// If the session does not exist, the gathering ID is empty and can be used
		if _, ok := Sessions[gatheringID]; !ok {
			return gatheringID
		}

		gatheringID++
	}

	return 0
}

// DeleteIndex removes a value from a slice with the given index
func DeleteIndex(s []uint32, index int) []uint32 {
	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

// FindOtherConnectionID searches a connection ID on the gathering that isn't the given one
func FindOtherConnectionID(myConnectionID uint32, gathering uint32) uint32 {
	for _, connectionID := range Sessions[gathering].ConnectionIDs {
		if connectionID != myConnectionID {
			return connectionID
		}
	}
	return 0
}

// RemoveConnectionIDFromRoom removes a client from the gathering
func RemoveConnectionIDFromRoom(clientConnectionID uint32, gathering uint32) {
	for index, connectionID := range Sessions[gathering].ConnectionIDs {
		if connectionID == clientConnectionID {
			Sessions[gathering].ConnectionIDs = DeleteIndex(Sessions[gathering].ConnectionIDs, index)
		}
	}
	if len(Sessions[gathering].ConnectionIDs) == 0 {
		delete(Sessions, gathering)
	}
}

// FindClientSession searches the gathering where the client is on
func FindClientSession(clientConnectionID uint32) uint32 {
	for gatheringID := range Sessions {
		for _, connectionID := range Sessions[gatheringID].ConnectionIDs {
			if connectionID == clientConnectionID {
				return gatheringID
			}
		}
	}
	return 0
}

// RemoveConnectionIDFromAllSessions removes a client from every session
func RemoveConnectionIDFromAllSessions(clientConnectionID uint32) {
	foundSession := FindClientSession(clientConnectionID)
	if foundSession != 0 {
		RemoveConnectionIDFromRoom(clientConnectionID, uint32(foundSession))
	}
}

// SearchGatheringWithMatchmakeSession finds a gathering that matches with a MatchmakeSession
func SearchGatheringWithMatchmakeSession(searchMatchmakeSession *match_making_types.MatchmakeSession) uint32 {
	var returnSessionIndex uint32 = 0

	// This portion finds any sessions that match the search session. It does not care about anything beyond that, such as if the match is already full. This is handled below.
	candidateSessionIndexes := make([]uint32, 0, len(Sessions))
	for index, session := range Sessions {
		if session.SearchMatchmakeSession.Equals(searchMatchmakeSession) {
			candidateSessionIndexes = append(candidateSessionIndexes, index)
		}
	}
	for _, sessionIndex := range candidateSessionIndexes {
		sessionToCheck := Sessions[sessionIndex]
		if len(sessionToCheck.ConnectionIDs) >= int(sessionToCheck.GameMatchmakeSession.MaximumParticipants) {
			continue
		} else {
			returnSessionIndex = sessionIndex //found a match
			break
		}
	}
	return returnSessionIndex
}

// SearchGatheringWithMatchmakeSession finds a gathering that matches with a MatchmakeSession
func SearchGatheringWithSearchCriteria(lstSearchCriteria []*match_making_types.MatchmakeSessionSearchCriteria) uint32 {
	var returnSessionIndex uint32 = 0

	// This portion finds any sessions that match the search session. It does not care about anything beyond that, such as if the match is already full. This is handled below.
	candidateSessionIndexes := make([]uint32, 0, len(Sessions))
	for index, session := range Sessions {
		if(len(lstSearchCriteria) == len(session.SearchCriteria)){
			for criteriaIndex, criteria := range session.SearchCriteria {
				if criteria.Equals(lstSearchCriteria[criteriaIndex]) {
					candidateSessionIndexes = append(candidateSessionIndexes, index)
				}
			}
		}
	}
	for _, sessionIndex := range candidateSessionIndexes {
		sessionToCheck := Sessions[sessionIndex]
		if len(sessionToCheck.ConnectionIDs) >= int(sessionToCheck.GameMatchmakeSession.MaximumParticipants) {
			continue
		} else {
			returnSessionIndex = sessionIndex //found a match
			break
		}
	}
	return returnSessionIndex
}
