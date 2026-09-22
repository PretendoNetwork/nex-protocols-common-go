package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func DeleteCommonData(manager *commonglobals.RankingManager, pid types.PID, uniqueID types.UInt32) *nex.Error {
	result, err := manager.Database.Exec(`
		UPDATE ranking_legacy.common_data
			SET deleted = true
		WHERE owner_pid = $1 AND unique_id = $2
	`, pid, uniqueID)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}
	rows, _ := result.RowsAffected()
	if rows < 1 {
		return nex.NewError(nex.ResultCodes.Ranking.NotFound, "No common data to delete")
	}

	return nil
}
