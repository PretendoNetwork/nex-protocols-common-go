package tracking

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// LogDisconnectGathering logs a gathering disconnect event on the given database
func LogDisconnectGathering(db *sql.DB, pid types.PID, gatheringID uint32, totalParticipants []uint64) *nex.Error {
	eventTime := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO tracking.disconnect_gathering (
		date,
		source_pid,
		gathering_id,
		total_participants
	) VALUES (
		$1,
		$2,
		$3,
		$4
	)`, eventTime, pid, gatheringID, pqextended.Array(totalParticipants))
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
