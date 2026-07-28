package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetTotalCount(manager *commonglobals.RankingManager, category types.UInt32) (types.UInt32, *nex.Error) {
	var result types.UInt32

	err := manager.Database.QueryRow(`
		SELECT COUNT(*) FROM ranking_legacy.scores
		WHERE category = $1 AND deleted = false
	`, category).Scan(&result)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return 0, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}

	return result, nil
}
