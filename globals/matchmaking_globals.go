package common_globals
import (
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	"math"
	"reflect"
)

type CommonMatchmakeSession struct {
	GameMatchmakeSession match_making.MatchmakeSession //used by the game, contains the current state of the MatchmakeSession
	SearchMatchmakeSession match_making.MatchmakeSession //used by the server when searching for matches, contains the state of the MatchmakeSession during the search process for easy compares
	ConnectionIDs []uint32 //players in the room, referenced by their connection IDs. This is used instead of the PID in order to ensure we're talking to the correct client (in case of e.g. multiple logins)
}

var Sessions map[uint32]*CommonMatchmakeSession
var CurrentGatheringID uint32

func DeleteIndex(s []uint32, index int) []uint32 {
    return append(s[:index], s[index+1:]...)
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

func FindClientSessions(clientConnectionID uint32) []uint32 {
	//TODO: is there any real instance where a client could be in multiple MatchmakeSessions at once? afaik there isn't, but I handle it regardless just in case (if we're sure there isn't, we could remove the handling and speed the code up)
	gatheringsFoundIn := make([]uint32, 0, len(Sessions))
	for gatheringIndex := range Sessions {
		for _, connectionID := range Sessions[gatheringIndex].ConnectionIDs {
			if connectionID == clientConnectionID {
				gatheringsFoundIn = append(gatheringsFoundIn, uint32(gatheringIndex))
			}
		}
	}
	return gatheringsFoundIn
}

func RemoveConnectionIDFromAllSessions(clientConnectionID uint32) {
	foundSessions := FindClientSessions(clientConnectionID)
	for i := 0; i < len(foundSessions); i++ {
		RemoveConnectionIDFromRoom(clientConnectionID, uint32(i))
	}
}

func FindSearchMatchmakeSession(searchMatchmakeSession match_making.MatchmakeSession) int {
	returnSessionIndex := math.MaxUint32
	//this portion finds any sessions that match the search session. It does not care about anything beyond that, such as if the match is already full. This is handled below.
	candidateSessionIndexes := make([]uint32, 0, len(Sessions))
	for index, session := range Sessions {
		if reflect.DeepEqual(session.SearchMatchmakeSession, searchMatchmakeSession) { // TODO - for Jon: Equals in StructureInterface
			candidateSessionIndexes = append(candidateSessionIndexes, index)
		}
	}
	for _, sessionIndex := range candidateSessionIndexes {
		sessionToCheck := Sessions[sessionIndex]
		if len(sessionToCheck.ConnectionIDs) >= int(sessionToCheck.GameMatchmakeSession.MaximumParticipants){
			continue
		} else {
			returnSessionIndex = int(sessionIndex) //found a match
			break
		}
	}
	return returnSessionIndex
}