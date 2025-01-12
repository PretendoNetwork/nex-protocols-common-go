package tracking

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// LogJoinGathering logs a gathering join event on the given database
func LogJoinGathering(db *sql.DB, sourcePID types.PID, gatheringID uint32, newParticipants []uint64, totalParticipants []uint64) *nex.Error {
	eventTime := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO tracking.join_gathering (
		date,
		source_pid,
		gathering_id,
		new_participants,
		total_participants
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5
	)`, eventTime, sourcePID, gatheringID, pqextended.Array(newParticipants), pqextended.Array(totalParticipants))
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
