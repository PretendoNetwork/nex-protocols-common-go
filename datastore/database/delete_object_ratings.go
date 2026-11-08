package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func DeleteObjectRatings(database common_globals.Database, dataID types.UInt64) *nex.Error {
	// * The lock created by deleteObjectRatingsInTransaction only exists until the end of the
	// * transaction, so if this isn't a transaction then we need to start one
	db, ok := database.(*sql.DB)
	if !ok {
		return deleteObjectRatingsInTransaction(database, dataID)
	}

	transaction, err := db.Begin()
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// * Does nothing once the transaction has been committed successfully
	defer transaction.Rollback()

	errCode := deleteObjectRatingsInTransaction(transaction, dataID)
	if errCode != nil {
		return errCode
	}

	if err := transaction.Commit(); err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
