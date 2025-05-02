package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func UpdateObjectReferenceData(manager *common_globals.DataStoreManager, dataID types.UInt64) *nex.Error {
	result, err := manager.Database.Exec(`
		UPDATE datastore.objects
		SET
			reference_count = reference_count + 1,
			last_reference_date = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
			expiration_date = CASE
				WHEN update_expiration_on_reference = true THEN (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') + (expiration_days * INTERVAL '1 day')
				ELSE expiration_date
			END
		WHERE data_id = $1;
	`, dataID)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	if affected, err := result.RowsAffected(); err != nil || affected == 0 {
		if err != nil {
			// TODO - Send more specific errors?
			return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		if affected == 0 {
			return nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
		}
	}

	return nil
}
