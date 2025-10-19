package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/tracking"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// JoinPersistentGathering joins participants from the same connection into a PersistentGathering. Returns the new number of participants
func JoinPersistentGathering(manager *common_globals.MatchmakingManager, persistentGathering match_making_types.PersistentGathering, connection *nex.PRUDPConnection, vacantParticipants uint16, joinMessage string) (uint32, *nex.Error) {
	newParticipants, nexError := match_making_database.JoinGathering(manager, uint32(persistentGathering.ID), connection, vacantParticipants, joinMessage)
	if nexError != nil {
		return 0, nexError
	}

	participationCount, nexError := UpdatePersistentGatheringParticipationCount(manager, connection.PID(), uint32(persistentGathering.ID))
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return newParticipants, nil
	}

	nexError = tracking.LogParticipateCommunity(manager.Database, connection.PID(), uint32(persistentGathering.ID), uint32(persistentGathering.ID), participationCount)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return newParticipants, nil
	}

	return newParticipants, nil
}
