package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func DeleteObjectRatings(database common_globals.Database, dataID types.UInt64) *nex.Error {
	// * The lock created by deleteObjectRatings only exists until the end of the
	// * transaction, so if this isn't a transaction then we need to start one
	db, ok := database.(*sql.DB)
	if !ok {
		return deleteObjectRatings(database, dataID)
	}

	transaction, err := db.Begin()
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// * Does nothing once the transaction has been committed successfully
	defer transaction.Rollback()

	errCode := deleteObjectRatings(transaction, dataID)
	if errCode != nil {
		return errCode
	}

	if err := transaction.Commit(); err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}

func deleteObjectRatings(database common_globals.Database, dataID types.UInt64) *nex.Error {
	errCode := LockRatingSlotsByDataIDs(database, types.List[types.UInt64]{dataID})
	if errCode != nil {
		return errCode
	}

	_, err := database.Exec(`
		WITH
			logs AS (
				DELETE FROM datastore.rating_logs WHERE data_id=$1
			),
			history AS (
				UPDATE datastore.rating_history SET removed = true WHERE data_id=$1 AND NOT removed
			)
		UPDATE datastore.rating_slots
		SET total_value = initial_value, rating_count = 0
		WHERE data_id=$1
	`, dataID)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// TODO - Do we care about result.RowsAffected() here?

	return nil
}
