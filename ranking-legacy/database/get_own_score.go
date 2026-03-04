package database

import (
	"database/sql"
	"errors"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetOwnScore(manager *commonglobals.RankingManager, pid types.PID, uniqueID types.UInt32, category types.UInt32) (types.List[types.UInt32], *nex.Error) {
	var result types.List[types.UInt32]
	err := manager.Database.QueryRow(`
		SELECT scores.scores FROM ranking_legacy.scores
		WHERE owner_pid = $1 AND unique_id = $2 AND category = $3 AND deleted = false
	`, pid, uniqueID, category).Scan(&result)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nex.NewError(nex.ResultCodes.Ranking.NotFound, "Own score not found")
	} else if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}

	return result, nil
}
