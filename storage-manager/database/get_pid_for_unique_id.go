package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetPidForUniqueId(manager *common_globals.StorageManagerManager, uniqueId types.UInt32) (types.PID, *nex.Error) {
	var pid types.PID
	err := manager.Database.QueryRow(
		`SELECT associated_pid FROM storage_manager.unique_ids WHERE unique_id = $1`, uniqueId,
	).Scan(&pid)
	if err != nil {
		return 0, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}

	return pid, nil
}
