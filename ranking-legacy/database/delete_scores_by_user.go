package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func DeleteScoresByUser(manager *commonglobals.RankingManager, pid types.PID, uniqueID types.UInt32) *nex.Error {
	result, err := manager.Database.Exec(`
		UPDATE ranking_legacy.scores SET deleted = true
		WHERE owner_pid = $1 AND unique_id = $2
	`, pid, uniqueID)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}
	rows, err := result.RowsAffected()
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}
	// Might be inaccurate to return an error here
	if rows < 1 {
		return nex.NewError(nex.ResultCodes.Ranking.NotFound, "No scores found for deletion")
	}

	return nil
}
