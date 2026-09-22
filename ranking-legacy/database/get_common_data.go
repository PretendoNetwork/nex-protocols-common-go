package database

import (
	"database/sql"
	"errors"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetCommonData(manager *commonglobals.RankingManager, pid types.PID, uniqueID types.UInt32) (types.Buffer, *nex.Error) {
	var result types.Buffer

	err := manager.Database.QueryRow(`
		SELECT data FROM ranking_legacy.common_data
		WHERE owner_pid = $1 AND unique_id = $2 AND deleted = false
	`, pid, uniqueID).Scan(&result)
	if errors.Is(err, sql.ErrNoRows) {
		return types.Buffer{}, nex.NewError(nex.ResultCodes.Ranking.NotFound, "Common data not found")
	} else if err != nil {
		return types.Buffer{}, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}

	return result, nil
}
