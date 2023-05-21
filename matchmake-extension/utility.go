package matchmake_extension

import (
	//nex "github.com/PretendoNetwork/nex-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	//"github.com/PretendoNetwork/plogger-go"
	"reflect"
	"math"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func FindSearchMatchmakeSession(searchMatchmakeSession match_making.MatchmakeSession) int {
	returnSessionIndex := math.MaxUint32
	//this portion finds any sessions that match the search session. It does not care about anything beyond that, such as if the match is already full. This is handled below.
	candidateSessionIndexes := []int{}
	for index, session := range common_globals.Sessions {
		if reflect.DeepEqual(session.SearchMatchmakeSession, searchMatchmakeSession) { // TODO - for Jon: Equals in StructureInterface
			candidateSessionIndexes = append(candidateSessionIndexes, index)
		}
	}
	for _, sessionIndex := range candidateSessionIndexes {
		sessionToCheck := common_globals.Sessions[sessionIndex]
		if len(sessionToCheck.ConnectionIDs) >= int(sessionToCheck.GameMatchmakeSession.MaximumParticipants){
			continue
		} else {
			returnSessionIndex = sessionIndex //found a match
			break
		}
	}
	return returnSessionIndex
}