package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

// UpdateSessionHost updates the owner and host PID of the session
func UpdateSessionHost(db *sql.DB, gatheringID uint32, ownerPID *types.PID, hostPID *types.PID) *nex.Error {
	result, err := db.Exec(`UPDATE matchmaking.gatherings SET owner_pid=$1, host_pid=$2 WHERE id=$3`, ownerPID.Value(), hostPID.Value(), gatheringID)
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
