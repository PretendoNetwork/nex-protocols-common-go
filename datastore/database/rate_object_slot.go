package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func RateObjectSlot(database common_globals.Database, manager *common_globals.DataStoreManager, dataID types.UInt64, slot types.UInt8, pid types.PID, value types.Int32, settings datastore_types.DataStoreRatingInitParam) *nex.Error {
	// * The lock created by rateObjectSlotInTransaction only exists until the end of the
	// * transaction, so if this isn't a transaction then we need to start one
	db, ok := database.(*sql.DB)
	if !ok {
		return rateObjectSlotInTransaction(database, manager, dataID, slot, pid, value, settings)
	}

	transaction, err := db.Begin()
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// * Does nothing once the transaction has been committed successfully
	defer transaction.Rollback()

	errCode := rateObjectSlotInTransaction(transaction, manager, dataID, slot, pid, value, settings)
	if errCode != nil {
		return errCode
	}

	if err := transaction.Commit(); err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
