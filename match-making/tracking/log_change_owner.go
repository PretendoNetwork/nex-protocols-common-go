package tracking

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

// LogChangeOwner logs an owner change event on the given database
func LogChangeOwner(db *sql.DB, sourcePID types.PID, gatheringID uint32, oldOwnerPID types.PID, newOwnerPID types.PID) *nex.Error {
	eventTime := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO tracking.change_owner (
		date,
		source_pid,
		gathering_id,
		old_owner_pid,
		new_owner_pid
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5
	)`, eventTime, sourcePID, gatheringID, oldOwnerPID, newOwnerPID)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
