package utility_database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

// InsertUniqueIDsByUser inserts a unique ID + password (if applicable) combination into the database, associated with the given user's PID
// isPrimary is used to indicate if the 0th unique ID in the array should be set as a primary ID (in most cases, this should be true)
func InsertUniqueIDsByUser(manager *common_globals.UtilityManager, userPID types.PID, uniqueIDInfos types.List[utility_types.UniqueIDInfo], isPrimary bool) *nex.Error {
	var err error

	if len(uniqueIDInfos) == 0 {
		common_globals.Logger.Error("Tried to pass in empty array to InsertUniqueIDsByUser!")
		return nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	currentTime := time.Now().UTC()

	for index, uniqueIDInfo := range uniqueIDInfos {
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
			uniqueIDInfo.NEXUniqueID,
			uniqueIDInfo.NEXUniqueIDPassword,
			userPID,
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
