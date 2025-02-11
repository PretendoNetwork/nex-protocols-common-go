package database

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// FindGatheringByID finds a gathering on a database with the given ID. Returns the gathering, its type, the participant list and the started time
func FindGatheringByID(manager *common_globals.MatchmakingManager, id uint32) (match_making_types.Gathering, string, []uint64, types.DateTime, *nex.Error) {
	row := manager.Database.QueryRow(`SELECT owner_pid, host_pid, min_participants, max_participants, participation_policy, policy_argument, flags, state, description, type, participants, started_time FROM matchmaking.gatherings WHERE id=$1 AND registered=true`, id)

	gathering := match_making_types.NewGathering()
	var gatheringType string
	var participants []uint64
	var startedTime time.Time

	err := row.Scan(
		&gathering.OwnerPID,
		&gathering.HostPID,
		&gathering.MinimumParticipants,
		&gathering.MaximumParticipants,
		&gathering.ParticipationPolicy,
		&gathering.PolicyArgument,
		&gathering.Flags,
		&gathering.State,
		&gathering.Description,
		&gatheringType,
		pqextended.Array(&participants),
		&startedTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return gathering, "", nil, types.NewDateTime(0), nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, err.Error())
		} else {
			return gathering, "", nil, types.NewDateTime(0), nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	gathering.ID = types.NewUInt32(id)
	startedDateTime := types.NewDateTime(0)

	return gathering, gatheringType, participants, startedDateTime.FromTimestamp(startedTime), nil
}
