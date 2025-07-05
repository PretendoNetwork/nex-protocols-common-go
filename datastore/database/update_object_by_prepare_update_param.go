package database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func UpdateObjectByPrepareUpdateParam(manager *common_globals.DataStoreManager, updateParam datastore_types.DataStorePrepareUpdateParam, currentData datastore_types.DataStoreMetaInfo) (types.UInt32, *nex.Error) {
	// * Max size is 10MiB.
	// * Does not seem to be a constant in DataStore, so just inline it for now?
	if updateParam.Size > 0xA00000 {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object larger than 10MiB")
	}

	needsReview := (currentData.Flag & types.UInt32(datastore_constants.DataFlagNeedReview)) != 0
	needUploadCompletion := (currentData.Flag & types.UInt32(datastore_constants.DataFlagNeedCompletion)) != 0

	status := currentData.Status
	if needsReview {
		status = types.UInt8(datastore_constants.DataStatusPending)
	}

	uploadCompleted := false
	if !needUploadCompletion {
		uploadCompleted = true
	}

	now := time.Now().UTC()
	expirationDate := time.Now().UTC().Add(time.Duration(currentData.Period) * 24 * time.Hour)

	var version types.UInt32

	// TODO - Do we want to update the row, or make a new row for each update version?
	//        Doing so would use more space, but would let us track individual updates
	err := manager.Database.QueryRow(`
		UPDATE datastore.objects
		SET
			version = version + 1,
			size = $1,
			extra_data = $2,
			status = $3,
			update_date = $4,
			true_update_date = $5,
			expiration_date = $6,
			upload_completed = $7
		WHERE data_id = $8
		RETURNING version
	`, updateParam.Size, pq.Array(updateParam.ExtraData), status, now, now, expirationDate, uploadCompleted, updateParam.DataID).Scan(&version)

	if err != nil {
		// TODO - Send more specific errors?
		return 0, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return version, nil
}
