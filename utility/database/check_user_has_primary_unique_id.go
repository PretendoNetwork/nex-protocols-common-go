package utility_database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// CheckUserHasPrimaryUniqueID checks if a given user already has a primary unique ID, and returns the result, as well as the ID if applicable
func CheckUserHasPrimaryUniqueID(manager *common_globals.UtilityManager, userPID types.PID) (bool, types.UInt64, *nex.Error) {
	primaryExists := true
	var primaryID types.UInt64

	err := manager.Database.QueryRow(`SELECT unique_id FROM utility.unique_ids WHERE associated_pid=$1 AND is_primary_id=true`,
		userPID,
	).Scan(
		&primaryID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			primaryExists = false
			primaryID = 0
		} else {
			return false, 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	return primaryExists, primaryID, nil
}
