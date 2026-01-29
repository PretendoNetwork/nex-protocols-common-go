package utility_database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

// GetUserAssociatedUniqueIDs gets all unique ID + password (if applicable) combinations associated with a user's PID
func GetUserAssociatedUniqueIDs(manager *common_globals.UtilityManager, userPID types.PID) (types.List[utility_types.UniqueIDInfo], *nex.Error) {
	rows, err := manager.Database.Query(`
			SELECT 
				unique_id,
				password 
			FROM utility.unique_ids
			WHERE associated_pid=$1 ORDER BY is_primary_id DESC, associated_time DESC`,
		userPID,
	)
	if err != nil {
		return types.List[utility_types.UniqueIDInfo]{}, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}
	defer rows.Close()

	uniqueIDInfos := make(types.List[utility_types.UniqueIDInfo], 0)
	for rows.Next() {
		uniqueIDInfo := utility_types.NewUniqueIDInfo()

		err = rows.Scan(
			&uniqueIDInfo.NEXUniqueID,
			&uniqueIDInfo.NEXUniqueIDPassword,
		)
		if err != nil {
			return types.List[utility_types.UniqueIDInfo]{}, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		uniqueIDInfos = append(uniqueIDInfos, uniqueIDInfo)
	}

	return uniqueIDInfos, nil
}
