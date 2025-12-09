package utility_database

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GenerateNEXUniqueID(manager *common_globals.UtilityManager, packet nex.PacketInterface) (types.UInt64, *nex.Error) {
	var uniqueID types.UInt64

	err := binary.Read(rand.Reader, binary.BigEndian, &uniqueID)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	primaryExists, _, nexError := CheckUserHasPrimaryUniqueID(manager, packet.Sender().PID())
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return 0, nexError
	}

	nexError = InsertUniqueIDsByUser(manager, packet.Sender().PID(), types.List[types.UInt64]{uniqueID}, primaryExists)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return 0, nexError
	}

	return uniqueID, nil
}
