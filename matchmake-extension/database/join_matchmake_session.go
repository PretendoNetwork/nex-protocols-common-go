package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// JoinMatchmakeSession joins participants from the same connection into a MatchmakeSession. Returns the new number of participants
func JoinMatchmakeSession(manager *common_globals.MatchmakingManager, matchmakeSession match_making_types.MatchmakeSession, connection *nex.PRUDPConnection, vacantParticipants uint16, joinMessage string) (uint32, *nex.Error) {
	newParticipants, nexError := match_making_database.JoinGathering(manager, uint32(matchmakeSession.ID), connection, vacantParticipants, joinMessage)
	if nexError != nil {
		return 0, nexError
	}

	return newParticipants, nil
}
