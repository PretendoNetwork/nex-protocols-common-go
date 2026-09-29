package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetPersistentGatheringParticipationCount gets the amount of persistent gatherings a user is participating
func GetPersistentGatheringParticipationCount(manager *common_globals.MatchmakingManager, sourcePID types.PID) (int, *nex.Error) {
	var count int
	err := manager.Database.QueryRow(`SELECT COUNT(*)
			FROM matchmaking.gatherings
			WHERE $1 = ANY(participants)`, uint64(sourcePID)).Scan(&count)
	if err != nil {
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return count, nil
}
