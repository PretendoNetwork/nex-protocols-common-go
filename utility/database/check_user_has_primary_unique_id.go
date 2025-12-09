package utility_database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func CheckUserHasPrimaryUniqueID(manager *common_globals.UtilityManager, userPid types.PID) (bool, types.UInt64, *nex.Error) {
	primaryExists := true
	var primaryId types.UInt64

	err := manager.Database.QueryRow(`SELECT unique_id FROM utility.unique_ids WHERE associated_pid=$1 AND is_primary_id=true`,
		userPid,
	).Scan(
		&primaryId,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			primaryExists = false
			primaryId = types.UInt64(0)
		} else {
			return false, 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	return primaryExists, primaryId, nil
}
