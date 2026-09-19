package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func UpdateObjectLock(database common_globals.Database, dataID types.UInt64, lockID types.UInt32) *nex.Error {
	var useReadLock bool
	var currentLockID types.UInt32

	err := database.QueryRow(`
		SELECT
			use_read_lock,
			COALESCE(lock_id, 0)
		FROM datastore.objects
		WHERE data_id=$1 AND deleted=false AND upload_completed=true
	`, dataID).Scan(&useReadLock, &currentLockID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// * Do nothing for objects without DATA_FLAG_USE_READ_LOCK set
	if !useReadLock {
		return nil
	}

	// * Wrong lock ID for a read-locked object
	if currentLockID != 0 && lockID != 0 && currentLockID != lockID {
		return nex.NewError(nex.ResultCodes.DataStore.AlreadyLocked, "change_error")
	}

	// * Either read-lock the object with the new lock ID, or release the lock
	// * if the provided lock ID is 0. I know this looks weird, but it's how
	// * the real servers work
	// * https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Lock_ID
	if currentLockID != lockID {
		_, err := database.Exec(`UPDATE datastore.objects SET lock_id = $1 WHERE data_id=$2`, lockID, dataID)
		if err != nil {
			// TODO - Send more specific errors?
			return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}
	}

	return nil
}
