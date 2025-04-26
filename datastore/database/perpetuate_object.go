package database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func PerpetuateObject(manager *common_globals.DataStoreManager, ownerPID types.PID, persistenceInitParam datastore_types.DataStorePersistenceInitParam, dataID uint64) *nex.Error {
	// * Assumes the slot is already available and has
	// * been cleared of any previous objects prior, and
	// * that the object is not in another slot already
	_, err := manager.Database.Exec(`INSERT INTO datastore.persistence_slots (
		pid,
		slot,
		data_id,
		delete_last_object
	) VALUES (
		$1,
		$2,
		$3,
		$4
	) ON CONFLICT (pid, slot) DO UPDATE SET
		data_id = EXCLUDED.data_id,
		delete_last_object = EXCLUDED.delete_last_object`,
		ownerPID,
		persistenceInitParam.PersistenceSlotID,
		dataID,
		persistenceInitParam.DeleteLastObject,
	)

	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	expirationDate := time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC)

	_, err = manager.Database.Exec(`UPDATE datastore.objects SET expiration_date = $1 WHERE data_id=$2`, expirationDate, dataID)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
