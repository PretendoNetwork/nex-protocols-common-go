package utility_database

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

// GenerateNEXUniqueIDWithPassword generates a unique ID, associated with the given user's PID, optionally adding a password
func GenerateNEXUniqueIDWithPassword(manager *common_globals.UtilityManager, userPID types.PID, usePassword bool) (utility_types.UniqueIDInfo, *nex.Error) {
	uniqueIDInfo := utility_types.NewUniqueIDInfo()

	err := binary.Read(rand.Reader, binary.NativeEndian, &uniqueIDInfo.NEXUniqueID)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return utility_types.UniqueIDInfo{}, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	if usePassword {
		err = binary.Read(rand.Reader, binary.NativeEndian, &uniqueIDInfo.NEXUniqueIDPassword)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return utility_types.UniqueIDInfo{}, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
		}
	}

	// As rare as this should be in the first place, I don't think calling it from itself should be a problem
	nexError := CheckUniqueIDAlreadyExists(manager, uniqueIDInfo)
	if nexError != nil && nexError.ResultCode == nex.ResultCodes.Core.SystemError {
		return GenerateNEXUniqueIDWithPassword(manager, userPID, usePassword)
	} else if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return utility_types.UniqueIDInfo{}, nexError
	}

	primaryExists, _, nexError := CheckUserHasPrimaryUniqueID(manager, userPID)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return utility_types.UniqueIDInfo{}, nexError
	}

	nexError = InsertUniqueIDsByUser(manager, userPID, types.List[utility_types.UniqueIDInfo]{uniqueIDInfo}, !primaryExists)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return utility_types.UniqueIDInfo{}, nexError
	}

	return uniqueIDInfo, nil
}
