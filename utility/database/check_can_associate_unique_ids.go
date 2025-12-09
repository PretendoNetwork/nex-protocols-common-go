package utility_database

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func CheckCanAssociateUniqueIDs(manager *common_globals.UtilityManager, userPid types.PID, uniqueIds, passwords types.List[types.UInt64]) *nex.Error {
	var uniqueId, dbPassword types.UInt64
	var associatedPid types.PID

	rows, err := manager.Database.Query(`SELECT unique_id, associated_pid, password FROM utility.unique_ids WHERE unique_id=ANY($1)`,
		uniqueIds,
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	rowCount := 0
	for rows.Next() {
		rowCount++

		err = rows.Scan(
			&uniqueId,
			&associatedPid,
			&dbPassword,
		)
		if err != nil {
			return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		// TODO - Is this a correct assumption?
		if associatedPid != userPid {
			return nex.NewError(nex.ResultCodes.Core.Unknown, "One of the unique ids is already owned by this user")
		}

		targetConnection := manager.Endpoint.FindConnectionByPID(uint64(associatedPid))
		if targetConnection != nil || !manager.AllowUniqueIDStealing {
			return nex.NewError(nex.ResultCodes.Core.AccessDenied, "Unique id stealing is disabled or the target is online")
		}

		index := slices.Index(uniqueIds, uniqueId)
		if index == -1 {
			return nex.NewError(nex.ResultCodes.Core.Unknown, "Index of unique id not found in array, this SHOULD NOT HAPPEN")
		}

		if dbPassword != passwords[index] {
			return nex.NewError(nex.ResultCodes.Core.AccessDenied, "Invalid password for a unique id")
		}
	}

	if rowCount != len(uniqueIds) {
		return nex.NewError(nex.ResultCodes.Core.InvalidArgument, "One or more of the provided unique ids do not exist")
	}

	return nil
}
