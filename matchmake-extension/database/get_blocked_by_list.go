package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetBlockedByList retrieves a list of PIDs who have blocked the given PID
func GetBlockedByList(manager *common_globals.MatchmakingManager, blockedPID types.PID) ([]types.PID, *nex.Error) {
	var userPIDs []types.PID

	rows, err := manager.Database.Query(`
		SELECT user_pid FROM matchmaking.block_lists WHERE blocked_pid = $1
	`, uint64(blockedPID))

	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var userPID types.PID
		if err := rows.Scan(&userPID); err != nil {
			common_globals.Logger.Error(err.Error())
			continue
		}
		userPIDs = append(userPIDs, userPID)
	}

	if err := rows.Err(); err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return userPIDs, nil
}
