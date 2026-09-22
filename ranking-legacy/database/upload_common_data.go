package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func UploadCommonData(manager *commonglobals.RankingManager, pid types.PID, uniqueID types.UInt32, commonData []byte) *nex.Error {
	_, err := manager.Database.Exec(`
		INSERT INTO ranking_legacy.common_data
    		(owner_pid, unique_id, data, create_time, update_time)
			VALUES ($1, $2, $3, now(), now())
    	ON CONFLICT (owner_pid, unique_id) DO UPDATE
    	    SET data = EXCLUDED.data, deleted = false, update_time = now()
	`, pid, uniqueID, commonData)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}

	return nil
}
