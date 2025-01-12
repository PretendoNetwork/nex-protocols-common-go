package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// UpdateSessionHost updates the owner and host PID of the session
func UpdateSessionHost(manager *common_globals.MatchmakingManager, gatheringID uint32, ownerPID types.PID, hostPID types.PID) *nex.Error {
	result, err := manager.Database.Exec(`UPDATE matchmaking.gatherings SET owner_pid=$1, host_pid=$2 WHERE id=$3`, ownerPID, hostPID, gatheringID)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	if rowsAffected == 0 {
		return nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
	}

	return nil
}
