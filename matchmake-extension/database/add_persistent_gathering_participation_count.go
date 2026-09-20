package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// UpdatePersistentGatheringParticipationCount marks a user as participating in a Persistent Gathering. Returns nothing
func AddPersistentGatheringParticipation(manager *common_globals.MatchmakingManager, userPID types.PID, gatheringID uint32) *nex.Error {
	_, err := manager.Database.Exec(`INSERT INTO matchmaking.community_participations (
		user_pid,
		gathering_id
	) VALUES (
		$1,
		$2
	) ON CONFLICT DO NOTHING`,
		uint64(userPID),
		gatheringID,
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
