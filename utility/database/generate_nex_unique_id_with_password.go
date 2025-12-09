package utility_database

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GenerateNEXUniqueIDWithPassword(manager *common_globals.UtilityManager, packet nex.PacketInterface) (uint64, uint64, *nex.Error) {
	var uniqueID uint64
	var password uint64

	err := binary.Read(rand.Reader, binary.BigEndian, &uniqueID)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return 0, 0, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	err = binary.Read(rand.Reader, binary.BigEndian, &password)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return 0, 0, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	primaryExists, _, nexError := CheckUserHasPrimaryUniqueID(manager, packet.Sender().PID())
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return 0, 0, nexError
	}

	nexError = InsertUniqueIDsByUserWithPasswords(manager, packet.Sender().PID(), []uint64{uniqueID}, []uint64{password}, primaryExists)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return 0, 0, nexError
	}

	return uniqueID, password, nil
}
