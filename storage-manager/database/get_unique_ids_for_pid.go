package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetUniqueIDsForPID(manager *commonglobals.StorageManagerManager, pid types.PID) (types.List[types.UInt32], *nex.Error) {
	rows, err := manager.Database.Query(
		`SELECT unique_id FROM storage_manager.unique_ids WHERE associated_pid = $1`, pid,
	)
	if err != nil {
		return types.NewList[types.UInt32](), nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}
	defer rows.Close()

	ids := types.NewList[types.UInt32]()
	for rows.Next() {
		var id types.UInt32
		err := rows.Scan(&id)
		if err != nil {
			return nil, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
		}

		ids = append(ids, id)
	}

	return ids, nil
}
