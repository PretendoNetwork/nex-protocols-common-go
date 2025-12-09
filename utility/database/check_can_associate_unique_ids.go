package utility_database

import (
	"slices"
	"strconv"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

func CheckCanAssociateUniqueIDs(manager *common_globals.UtilityManager, userPid types.PID, uniqueIds, passwords []uint64) *nex.Error {
	var associatedPid, passwordDb, uniqueId uint64
	var associatedPidString, passwordString, uniqueIdString string

	rows, err := manager.Database.Query(`SELECT unique_id, associated_pid, password FROM utility.unique_ids WHERE unique_id=ANY($1)`,
		pqextended.UInt64Array(uniqueIds),
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	for rows.Next() {
		err = rows.Scan(
			&associatedPidString,
			&passwordString,
			&uniqueIdString,
		)
		if err != nil {
			return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		associatedPid, err = strconv.ParseUint(associatedPidString, 10, 64)
		if err != nil {
			return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		passwordDb, err = strconv.ParseUint(passwordString, 10, 64)
		if err != nil {
			return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		uniqueId, err = strconv.ParseUint(uniqueIdString, 10, 64)
		if err != nil {
			return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		// TODO - Is this a correct assumption?
		if associatedPid != uint64(userPid) {
			return nex.NewError(nex.ResultCodes.Core.Unknown, "A unique id is already owned by this user")
		}

		targetConnection := manager.Endpoint.FindConnectionByPID(associatedPid)
		if targetConnection != nil || !manager.AllowUniqueIDStealing {
			return nex.NewError(nex.ResultCodes.Core.AccessDenied, "Unique id stealing is disabled or the target is online")
		}

		index := slices.Index(uniqueIds, uniqueId)
		if index == -1 {
			return nex.NewError(nex.ResultCodes.Core.Unknown, "Index of unique id not found in array, this SHOULD NOT HAPPEN")
		}

		if passwordDb != passwords[index] {
			return nex.NewError(nex.ResultCodes.Core.AccessDenied, "Invalid password for a unique id")
		}
	}

	return nil
}
