package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetBlockList retrieves the block list for a given user PID
func GetBlockList(manager *common_globals.MatchmakingManager, userPID types.PID) ([]types.PID, *nex.Error) {
	var blockedPIDs []types.PID

	rows, err := manager.Database.Query(`
		SELECT blocked_pid FROM matchmaking.block_lists WHERE user_pid = $1
	`, uint64(userPID))

	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var blockedPID types.PID
		if err := rows.Scan(&blockedPID); err != nil {
			common_globals.Logger.Error(err.Error())
			continue
		}
		blockedPIDs = append(blockedPIDs, blockedPID)
	}

	if err := rows.Err(); err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return blockedPIDs, nil
}
