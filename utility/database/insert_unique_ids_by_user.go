package utility_database

import (
	"strconv"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// isPrimary is used to indicate if the 0th unique id in the array should be set as a primary id
func InsertUniqueIDsByUser(manager *common_globals.UtilityManager, userPid types.PID, uniqueIds []uint64, isPrimary bool) *nex.Error {
	var err error
	if len(uniqueIds) == 0 {
		common_globals.Logger.Error("Tried to pass in empty array to InsertUniqueIDsByUser!")
		return nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	currentTime := time.Now().UTC()
	userPidString := strconv.FormatUint(uint64(userPid), 10)

	for index, uniqueId := range uniqueIds {
		_, err = manager.Database.Exec(`INSERT INTO utility.unique_ids (
			unique_id,
			password,
			associated_pid,
			associated_time,
			creation_time,
			is_primary_id
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6
		)`,
			strconv.FormatUint(uniqueId, 10),
			"0", // I hate this, but thats just how numerics work
			userPidString,
			currentTime,
			currentTime,
			index == 0 && isPrimary,
		)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
		}
	}

	return nil
}
