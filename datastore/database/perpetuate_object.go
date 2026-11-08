package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func PerpetuateObject(database common_globals.Database, ownerPID types.PID, slot types.UInt16, dataID uint64) *nex.Error {
	// * Assumes the slot is already available and has
	// * been cleared of any previous objects prior, and
	// * that the object is not in another slot already
	_, err := database.Exec(`INSERT INTO datastore.persistence_slots (
		pid,
		slot,
		data_id
	) VALUES (
		$1,
		$2,
		$3
	) ON CONFLICT (pid, slot) DO UPDATE SET data_id = EXCLUDED.data_id`,
		ownerPID,
		slot,
		dataID,
	)

	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	_, err = database.Exec(`UPDATE datastore.objects SET expiration_date = $1 WHERE data_id=$2`, types.NewPermanentDateTime(), dataID)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
