package database

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// FindGatheringByID finds a gathering on a database with the given ID. Returns the gathering, its type, the participant list and the started time
func FindGatheringByID(db *sql.DB, id uint32) (*match_making_types.Gathering, string, []uint64, *types.DateTime, *nex.Error) {
	row := db.QueryRow(`SELECT owner_pid, host_pid, min_participants, max_participants, participation_policy, policy_argument, flags, state, description, type, participants, started_time FROM matchmaking.gatherings WHERE id=$1 AND registered=true`, id)

	var ownerPID uint64
	var hostPID uint64
	var minParticipants uint16
	var maxParticipants uint16
	var participationPolicy uint32
	var policyArgument uint32
	var flags uint32
	var state uint32
	var description string
	var gatheringType string
	var participants []uint64
	var startedTime time.Time

	err := row.Scan(
		&ownerPID,
		&hostPID,
		&minParticipants,
		&maxParticipants,
		&participationPolicy,
		&policyArgument,
		&flags,
		&state,
		&description,
		&gatheringType,
		pqextended.Array(&participants),
		&startedTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", nil, nil, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, err.Error())
		} else {
			return nil, "", nil, nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	gathering := match_making_types.NewGathering()
	gathering.ID.Value = id
	gathering.OwnerPID = types.NewPID(ownerPID)
	gathering.HostPID = types.NewPID(hostPID)
	gathering.MinimumParticipants.Value = minParticipants
	gathering.MaximumParticipants.Value = maxParticipants
	gathering.ParticipationPolicy.Value = participationPolicy
	gathering.PolicyArgument.Value = policyArgument
	gathering.Flags.Value = flags
	gathering.State.Value = state
	gathering.Description.Value = description

	return gathering, gatheringType, participants, types.NewDateTime(0).FromTimestamp(startedTime), nil
}
