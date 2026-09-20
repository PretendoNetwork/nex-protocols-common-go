package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// RemovePersistentGatheringParticipation removes a user's participation from a persistent gathering. Returns nothing
func RemovePersistentGatheringParticipation(manager *common_globals.MatchmakingManager, userPID types.PID, gatheringID uint32) *nex.Error {
	_, err := manager.Database.Exec(`DELETE FROM matchmaking.community_participations WHERE user_pid = $1 AND gathering_id = $2`,
		uint64(userPID),
		gatheringID,
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
