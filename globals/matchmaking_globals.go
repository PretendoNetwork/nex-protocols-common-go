package common_globals
import (
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	"math"
)

type CommonMatchmakeSession struct {
	GameMatchmakeSession *match_making.MatchmakeSession //used by the game, contains the current state of the MatchmakeSession
	SearchMatchmakeSession *match_making.MatchmakeSession //used by the server when searching for matches, contains the state of the MatchmakeSession during the search process for easy compares
	ConnectionIDs []uint32 //players in the room, referenced by their connection IDs. This is used instead of the PID in order to ensure we're talking to the correct client (in case of e.g. multiple logins)
}

var Sessions map[uint32]*CommonMatchmakeSession
var CurrentGatheringID uint32

func DeleteIndex(s []uint32, index int) []uint32 {
	if len(s) <= 1 {
		return make([]uint32, 0)
	}

	return append(s[:index], s[index+1:]...)
}

func FindOtherConnectionID(myConnectionID uint32, gathering uint32) uint32 {
	for _, connectionID := range Sessions[gathering].ConnectionIDs {
		if connectionID != myConnectionID {
			return connectionID
		}
	}
	return math.MaxUint32
}

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

func FindClientSession(clientConnectionID uint32) uint32 {
	for gatheringID := range Sessions {
		for _, connectionID := range Sessions[gatheringID].ConnectionIDs {
			if connectionID == clientConnectionID {
				return gatheringID
			}
		}
	}
	return math.MaxUint32
}

func RemoveConnectionIDFromAllSessions(clientConnectionID uint32) {
	foundSession := FindClientSession(clientConnectionID)
	if foundSession != math.MaxUint32 {
		RemoveConnectionIDFromRoom(clientConnectionID, foundSession)
	}
}

func FindSearchMatchmakeSession(searchMatchmakeSession *match_making.MatchmakeSession) int {
	returnSessionIndex := math.MaxUint32
	//this portion finds any sessions that match the search session. It does not care about anything beyond that, such as if the match is already full. This is handled below.
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
			returnSessionIndex = int(sessionIndex) //found a match
			break
		}
	}
	return returnSessionIndex
}
