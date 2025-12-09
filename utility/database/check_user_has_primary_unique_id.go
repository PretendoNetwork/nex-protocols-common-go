package utility_database

import (
	"database/sql"
	"strconv"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func CheckUserHasPrimaryUniqueID(manager *common_globals.UtilityManager, userPid types.PID) (bool, uint64, *nex.Error) {
	var primaryExists bool
	var primaryId uint64
	var primaryIdString string

	err := manager.Database.QueryRow(`SELECT unique_id FROM utility.unique_ids WHERE associated_pid=$1 AND is_primary_id=true`,
		userPid,
	).Scan(
		&primaryIdString,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			primaryExists = false
			primaryId = 0
		} else {
			return false, 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	primaryId, err = strconv.ParseUint(primaryIdString, 10, 64)
	if err != nil {
		return false, 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return primaryExists, primaryId, nil
}
