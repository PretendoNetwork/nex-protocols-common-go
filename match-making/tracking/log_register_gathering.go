package tracking

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

// LogRegisterGathering logs a gathering registration event on the given database
func LogRegisterGathering(db *sql.DB, sourcePID types.PID, gatheringID uint32) *nex.Error {
	eventTime := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO tracking.register_gathering (
		date,
		source_pid,
		gathering_id
	) VALUES (
		$1,
		$2,
		$3
	)`, eventTime, sourcePID, gatheringID)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
