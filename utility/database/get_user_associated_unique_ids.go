package utility_database

import (
	"strconv"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetUserAssociatedUniqueIDs(manager *common_globals.UtilityManager, userPid types.PID) ([]uint64, []uint64, *nex.Error) {
	var password, uniqueId uint64
	var passwordString, uniqueIdString string

	uniqueIds := make([]uint64, 0)
	passwords := make([]uint64, 0)

	rows, err := manager.Database.Query(`
			SELECT 
				unique_id,
				password 
			FROM utility.unique_ids
			WHERE associated_pid=$1 ORDER BY is_primary_id DESC, associated_time DESC`,
		userPid,
	)
	if err != nil {
		return nil, nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	for rows.Next() {
		err = rows.Scan(
			&passwordString,
			&uniqueIdString,
		)
		if err != nil {
			return nil, nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		password, err = strconv.ParseUint(passwordString, 10, 64)
		if err != nil {
			return nil, nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		uniqueId, err = strconv.ParseUint(uniqueIdString, 10, 64)
		if err != nil {
			return nil, nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		uniqueIds = append(uniqueIds, uniqueId)
		passwords = append(passwords, password)
	}

	return uniqueIds, passwords, nil
}
