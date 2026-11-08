package utility_database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// ClearPIDUniqueIDAssociations clears all unique ID associations for a given user
func ClearPIDUniqueIDAssociations(manager *common_globals.UtilityManager, userPID types.PID) *nex.Error {
	_, err := manager.Database.Exec(`UPDATE utility.unique_ids SET associated_pid=0, associated_time=null, is_primary_id=false WHERE associated_pid=$1`, userPID)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	// Rows affected shouldn't matter, as long as there are no IDs associated with the PID it's fine
	return nil
}
