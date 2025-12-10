package utility_database

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

// GenerateNEXUniqueID generates a unique ID, associated with the given user's PID
func GenerateNEXUniqueID(manager *common_globals.UtilityManager, userPID types.PID) (types.UInt64, *nex.Error) {
	uniqueIDInfo := utility_types.NewUniqueIDInfo()

	err := binary.Read(rand.Reader, binary.NativeEndian, &uniqueIDInfo.NEXUniqueID)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	// As rare as this should be in the first place, I don't think calling it from itself should be a problem
	nexError := CheckUniqueIDAlreadyExists(manager, uniqueIDInfo)
	if nexError != nil && nexError.ResultCode == nex.ResultCodes.Core.SystemError {
		return GenerateNEXUniqueID(manager, userPID)
	} else if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return 0, nexError
	}

	primaryExists, _, nexError := CheckUserHasPrimaryUniqueID(manager, userPID)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return 0, nexError
	}

	nexError = InsertUniqueIDsByUser(manager, userPID, types.List[types.UInt64]{uniqueIDInfo.NEXUniqueID}, types.List[types.UInt64]{0}, !primaryExists)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return 0, nexError
	}

	return uniqueIDInfo.NEXUniqueID, nil
}
