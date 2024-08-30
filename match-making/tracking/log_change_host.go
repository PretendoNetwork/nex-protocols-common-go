package tracking

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

// LogChangeHost logs a host change event on the given database
func LogChangeHost(db *sql.DB, sourcePID *types.PID, gatheringID uint32, oldHostPID *types.PID, newHostPID *types.PID) *nex.Error {
	eventTime := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO tracking.change_host (
		date,
		source_pid,
		gathering_id,
		old_host_pid,
		new_host_pid
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5
	)`, eventTime, sourcePID.Value(), gatheringID, oldHostPID.Value(), newHostPID.Value())
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
