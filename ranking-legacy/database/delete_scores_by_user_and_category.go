package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func DeleteScoresByUserAndCategory(manager *commonglobals.RankingManager, pid types.PID, uniqueID types.UInt32, category types.UInt32) *nex.Error {
	result, err := manager.Database.Exec(`
		UPDATE ranking_legacy.scores SET deleted = true
		WHERE owner_pid = $1 AND unique_id = $2 AND category = $3
	`, pid, uniqueID, category)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}
	rows, err := result.RowsAffected()
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}
	if rows < 1 {
		return nex.NewError(nex.ResultCodes.Ranking.NotFound, "No scores found for deletion")
	}

	return nil
}
