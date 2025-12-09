package utility_database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// isPrimary is used to indicate if the 0th unique id in the array should be set as a primary id
func UpdateUniqueIDAssociations(manager *common_globals.UtilityManager, userPid types.PID, uniqueIds types.List[types.UInt64], isPrimary bool) *nex.Error {
	var err error

	if len(uniqueIds) == 0 {
		common_globals.Logger.Error("Tried to pass in empty array to UpdateUniqueIDAssociationsWithPasswords!")
		return nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	currentTime := time.Now().UTC()

	for index, uniqueId := range uniqueIds {
		_, err = manager.Database.Exec(`UPDATE utility.unique_ids SET 
				associated_pid=$2, 
				associated_time=$3, 
				is_primary_id=$4
			WHERE unique_id=$1
		`,
			uniqueId,
			userPid,
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
