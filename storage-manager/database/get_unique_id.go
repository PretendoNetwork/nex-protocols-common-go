package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetUniqueId returns the unique ID for a user, slot, card combo, or generates a new one if not present.
func GetUniqueId(manager *common_globals.StorageManagerManager, slotId types.UInt8, cardId int64, userPID types.PID) (types.UInt32, types.Bool, *nex.Error) {
	var card any = cardId
	if cardId < 0 {
		card = nil
	}

	var uniqueId types.UInt32
	var firstTime types.Bool

	// https://stackoverflow.com/a/74057102
	err := manager.Database.QueryRow(`
		/* Attempt to insert the new row first... */
		WITH new_id AS (
			INSERT INTO storage_manager.unique_ids
				(slot_id, card_id, associated_pid, associated_time, creation_time)
				VALUES
				($1, $2, $3, now(), now())
				ON CONFLICT DO NOTHING
				RETURNING unique_id, true AS first_time
		)
		SELECT * FROM new_id
		/* Otherwise union in the existing one */
		UNION ALL
		SELECT unique_id, false AS first_time FROM storage_manager.unique_ids
				 WHERE slot_id = $1
				   AND card_id = $2
				   AND associated_pid = $3
		LIMIT 1;
	`, slotId, card, userPID).Scan(
		&uniqueId,
		&firstTime,
	)
	if err != nil {
		return 0, false, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}

	return uniqueId, firstTime, nil
}
