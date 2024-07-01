package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// RegisterGathering registers a new gathering on the databse. No participants are added
func RegisterGathering(db *sql.DB, pid *types.PID, gathering *match_making_types.Gathering, gatheringType string) (*types.DateTime, *nex.Error) {
	startedTime := types.NewDateTime(0).Now()

	err := db.QueryRow(`INSERT INTO matchmaking.gatherings (
		owner_pid,
		host_pid,
		min_participants,
		max_participants,
		participation_policy,
		policy_argument,
		flags,
		state,
		description,
		type,
		started_time
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7,
		$8,
		$9,
		$10,
		$11
	) RETURNING id`,
		pid.Value(),
		pid.Value(),
		gathering.MinimumParticipants.Value,
		gathering.MaximumParticipants.Value,
		gathering.ParticipationPolicy.Value,
		gathering.PolicyArgument.Value,
		gathering.Flags.Value,
		gathering.State.Value,
		gathering.Description.Value,
		gatheringType,
		startedTime.Standard(),
	).Scan(&gathering.ID.Value)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	gathering.OwnerPID = pid.Copy().(*types.PID)
	gathering.HostPID = pid.Copy().(*types.PID)

	return startedTime, nil
}
