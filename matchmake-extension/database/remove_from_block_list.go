package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// RemoveFromBlockList removes a list of PIDs from a user's blocklist
func RemoveFromBlockList(manager *common_globals.MatchmakingManager, userPID types.PID, pidsToUnblock types.List[types.PID]) *nex.Error {

	pids := make([]uint64, len(pidsToUnblock))
	for i, pid := range pidsToUnblock {
		pids[i] = uint64(pid)
	}

	_, err := manager.Database.Exec(`
		DELETE FROM matchmaking.block_lists 
		WHERE user_pid = $1 AND blocked_pid = ANY($2)
	`, uint64(userPID), pqextended.Array(pids))

	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
