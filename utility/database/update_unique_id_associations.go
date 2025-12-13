package utility_database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

// UpdateUniqueIDAssociations updates the associations for a unique ID in the database based on the given user's PID (this does not
// check if the user should be able to associate the unique ID, see CheckCanAssociateUniqueIDs)
// isPrimary is used to indicate if the 0th unique ID in the array should be set as a primary ID (in most cases, this should be true)
func UpdateUniqueIDAssociations(manager *common_globals.UtilityManager, userPID types.PID, uniqueIDInfos types.List[utility_types.UniqueIDInfo], isPrimary bool) *nex.Error {
	var err error

	if len(uniqueIDInfos) == 0 {
		common_globals.Logger.Error("Tried to pass in empty array to UpdateUniqueIDAssociations!")
		return nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	currentTime := time.Now().UTC()
	uniqueIDs := make(types.List[types.UInt64], 0)
	for _, uniqueIDInfo := range uniqueIDInfos {
		uniqueIDs = append(uniqueIDs, uniqueIDInfo.NEXUniqueID)
	}

	_, err = manager.Database.Exec(`UPDATE utility.unique_ids SET 
			associated_pid=$2, 
			associated_time=$3, 
			is_primary_id=(CASE WHEN unique_id=$4 AND $5=true THEN true ELSE false)
		WHERE unique_id=ANY($1)
	`,
		uniqueIDs,
		userPID,
		currentTime,
		uniqueIDs[0],
		isPrimary,
	)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	return nil
}
