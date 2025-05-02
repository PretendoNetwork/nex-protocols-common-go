package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func GetObjectPasswords(manager *common_globals.DataStoreManager, caller types.PID, dataIDs types.List[types.UInt64]) (types.List[datastore_types.DataStorePasswordInfo], types.List[types.QResult], *nex.Error) {
	passwordInfos := types.NewList[datastore_types.DataStorePasswordInfo]()
	results := types.NewList[types.QResult]()

	if len(dataIDs) == 0 {
		return passwordInfos, results, nil
	}

	// * Return a row even if the data_id is invalid
	// * or if the caller is not the owner. Validated
	// * later
	rows, err := manager.Database.Query(`
		SELECT
			input.data_id,
			COALESCE(o.owner, 0) AS owner,
			COALESCE(o.access_password, 0) AS access_password,
			COALESCE(o.update_password, 0) AS update_password,
			CASE
				WHEN o.data_id IS NULL THEN $1 -- data_id is invalid
				WHEN o.owner != $2 THEN $3 -- caller does not own the object
				ELSE $4 -- all is well
			END AS result_code
		FROM unnest($5::bigint[]) AS input(data_id)
		LEFT JOIN datastore.objects o ON input.data_id = o.data_id
		ORDER BY input.data_id
	`,
		nex.ResultCodes.DataStore.NotFound,
		caller,
		nex.ResultCodes.DataStore.OperationNotAllowed,
		nex.ResultCodes.DataStore.Unknown, // * Used here to indicate a success
		pq.Array(dataIDs),
	)
	if err != nil {
		return passwordInfos, results, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		var dataID types.UInt64
		var owner types.PID
		var accessPassword types.UInt64
		var updatePassword types.UInt64
		var resultCode int

		err := rows.Scan(&dataID, &owner, &accessPassword, &updatePassword, &resultCode)
		if err != nil {
			return passwordInfos, results, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		passwordInfo := datastore_types.NewDataStorePasswordInfo()
		passwordInfo.DataID = dataID

		var result types.QResult

		if resultCode == int(nex.ResultCodes.DataStore.Unknown) {
			result = types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown)
			passwordInfo.AccessPassword = accessPassword
			passwordInfo.UpdatePassword = updatePassword
		} else {
			result = types.NewQResultError(uint32(resultCode))
			passwordInfo.AccessPassword = 0
			passwordInfo.UpdatePassword = 0
		}

		passwordInfos = append(passwordInfos, passwordInfo)
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return passwordInfos, results, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return passwordInfos, results, nil
}
