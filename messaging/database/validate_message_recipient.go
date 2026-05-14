package database

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_constants "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/constants"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
)

// ValidateMessageRecipient checks that the given PID can access the given recipient data
func ValidateMessageRecipient(manager *common_globals.MessagingManager, pid types.PID, recipientID types.UInt64, recipientType messaging_constants.RecipientType) bool {
	switch recipientType {
	case 1: // * PID - Valid if the recipient ID is the same as the given PID
		if pid != types.PID(recipientID) {
			return false
		}

		return true
	case 2: // * Gathering ID - Valid if the given PID is a participant on the gathering
		if manager.MatchmakingManager == nil {
			common_globals.Logger.Warning("MessagingManager.MatchmakingManager is not set!")
			return false
		}

		// * Check that the gathering exists
		manager.MatchmakingManager.Mutex.RLock()
		defer manager.MatchmakingManager.Mutex.RUnlock()
		_, _, participants, _, nexError := match_making_database.FindGatheringByID(manager.MatchmakingManager, uint32(recipientID))
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			return false
		}

		if !slices.Contains(participants, uint64(pid)) {
			return false
		}

		return true
	}

	common_globals.Logger.Errorf("Invalid recipient type %d", recipientType)
	return false
}
