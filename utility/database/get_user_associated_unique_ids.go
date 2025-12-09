package utility_database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetUserAssociatedUniqueIDs(manager *common_globals.UtilityManager, userPid types.PID) ([]uint64, []uint64, *nex.Error) {
	var uniqueId, password types.UInt64

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
			&uniqueId,
			&password,
		)
		if err != nil {
			return nil, nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		uniqueIds = append(uniqueIds, uint64(uniqueId))
		passwords = append(passwords, uint64(password))
	}

	return uniqueIds, passwords, nil
}
