package database

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
)

// CheckGatheringForParticipation checks that the given PIDs are participating on the gathering ID
func CheckGatheringForParticipation(manager *common_globals.MatchmakingManager, gatheringID uint32, participantsCheck []types.PID) *nex.Error {
	_, _, participants, _, err := database.FindGatheringByID(manager, gatheringID)
	if err != nil {
		return err
	}

	for _, participant := range participantsCheck {
		if !slices.Contains(participants, uint64(participant)) {
			return nex.NewError(nex.ResultCodes.RendezVous.NotParticipatedGathering, "change_error")
		}
	}

	return nil
}
