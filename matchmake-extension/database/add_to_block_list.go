package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// AddToBlockList adds a list of PIDs to a user's blocklist
func AddToBlockList(manager *common_globals.MatchmakingManager, userPID types.PID, pidsToBlock types.List[types.PID]) *nex.Error {
	tx, err := manager.Database.Begin()
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	stmt, err := tx.Prepare(`INSERT INTO matchmaking.block_lists (user_pid, blocked_pid) VALUES ($1, $2) ON CONFLICT (user_pid, blocked_pid) DO NOTHING`)
	if err != nil {
		tx.Rollback()
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}
	defer stmt.Close()

	for _, pidToBlock := range pidsToBlock {
		if _, err := stmt.Exec(uint64(userPID), uint64(pidToBlock)); err != nil {
			tx.Rollback()
			return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	if err := tx.Commit(); err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
