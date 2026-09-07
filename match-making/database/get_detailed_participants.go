package database

import (
	"database/sql"

	"github.com/PretendoNetwork/pq-extended"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	matchmaking_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// GetDetailedParticipants returns detailed information about the participants of the given gathering
func GetDetailedParticipants(manager *common_globals.MatchmakingManager, gatheringID types.UInt32, sourcePID types.PID) (types.List[matchmaking_types.ParticipantDetails], *nex.Error) {
	var participantList []uint64

	err := manager.Database.QueryRow(`SELECT participants FROM matchmaking.gatherings WHERE id = $1`, gatheringID).Scan(pqextended.Array(&participantList))
	if err != nil {
		if err == sql.ErrNoRows {
			return types.NewList[matchmaking_types.ParticipantDetails](), nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, err.Error())
		} else {
			return types.NewList[matchmaking_types.ParticipantDetails](), nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	if len(participantList) == 0 {
		// * Empty gathering
		return types.NewList[matchmaking_types.ParticipantDetails](), nil
	}

	if !slices.Contains(participantList, uint64(sourcePID)) {
		// * Is this the right error to use?
		return types.NewList[matchmaking_types.ParticipantDetails](), nex.NewError(nex.ResultCodes.RendezVous.NotParticipatedGathering, "User attempted GetDetailedParticipants inside of a NotParticipatedGathering")
	}

	var participantDetails types.List[matchmaking_types.ParticipantDetails]

	for _, participant := range common_globals.RemoveDuplicates(participantList) {
		participantInfo := matchmaking_types.NewParticipantDetails()

		for _, pid := range participantList {
			if participant == pid {
				participantInfo.UIParticipants += 1
			}
		}

		err = manager.Database.QueryRow(`SELECT pid, message FROM matchmaking.join_messages WHERE gathering_id = $1 AND pid = $2`, gatheringID, participant).Scan(&participantInfo.IDParticipant, &participantInfo.StrMessage)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			continue
		}

		accountDetails, err := manager.Endpoint.AccountDetailsByPID(types.NewPID(participant))
		if err != nil {
			common_globals.Logger.Error(err.Error())
			continue
		}

		participantInfo.StrName = types.String(accountDetails.Username)

		participantDetails = append(participantDetails, participantInfo)
	}

	return participantDetails, nil
}
