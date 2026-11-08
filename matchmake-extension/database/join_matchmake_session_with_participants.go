package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// JoinMatchmakeSessionWithParticipants joins participants into a gathering. Returns the new number of participants
func JoinMatchmakeSessionWithParticipants(manager *common_globals.MatchmakingManager, matchmakeSession match_making_types.MatchmakeSession, connection *nex.PRUDPConnection, additionalParticipants []types.PID, joinMessage string, joinMatchmakeSessionBehavior constants.JoinMatchmakeSessionBehavior) (uint32, *nex.Error) {
	newParticipants, nexError := match_making_database.JoinGatheringWithParticipants(manager, uint32(matchmakeSession.ID), connection, additionalParticipants, joinMessage, joinMatchmakeSessionBehavior)
	if nexError != nil {
		return 0, nexError
	}

	return newParticipants, nil
}
