package database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// GetOfficialCommunities returns the official communities based on the given parameters
func GetOfficialCommunities(manager *common_globals.MatchmakingManager, sourcePID types.PID, isAvailableOnly bool, resultRange types.ResultRange) ([]match_making_types.PersistentGathering, *nex.Error) {
	persistentGatherings := make([]match_making_types.PersistentGathering, 0)
	currentTime := time.Now().UTC()
	timeNever := types.NewDateTime(0).Standard()
	rows, err := manager.Database.Query(`SELECT
		g.id,
		g.owner_pid,
		g.host_pid,
		g.min_participants,
		g.max_participants,
		g.participation_policy,
		g.policy_argument,
		g.flags,
		g.state,
		g.description,
		pg.community_type,
		pg.password,
		pg.attribs,
		pg.application_buffer,
		pg.participation_start_date,
		pg.participation_end_date,
		(SELECT COUNT(ms.id)
			FROM matchmaking.matchmake_sessions AS ms
			INNER JOIN matchmaking.gatherings AS gms ON ms.id = gms.id
			WHERE gms.registered=true
			AND ms.matchmake_system_type=5 -- matchmake_system_type=5 is only used in matchmake sessions attached to a persistent gathering
			AND ms.attribs[1]=g.id) AS matchmake_session_count,
		COALESCE((SELECT cp.participation_count
			FROM matchmaking.community_participations AS cp
			WHERE cp.user_pid=$5
			AND cp.gathering_id=g.id), 0) AS participation_count
		FROM matchmaking.gatherings AS g
		INNER JOIN matchmaking.persistent_gatherings AS pg ON g.id = pg.id
		WHERE
		g.registered=true AND
		g.type='PersistentGathering' AND
		pg.community_type=2 AND
		(CASE WHEN $1 THEN
			(CASE WHEN pg.participation_start_date <> $6 THEN $2 >= pg.participation_start_date ELSE true END)
			AND
			(CASE WHEN pg.participation_end_date <> $6 THEN $2 <= pg.participation_end_date ELSE true END)
			ELSE true END)
		LIMIT $3 OFFSET $4`,
		isAvailableOnly,
		currentTime,
		uint32(resultRange.Length),
		uint32(resultRange.Offset),
		sourcePID,
		timeNever,
	)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	for rows.Next() {
		resultPersistentGathering := match_making_types.NewPersistentGathering()
		var resultAttribs []uint32
		var resultParticipationStartDate time.Time
		var resultParticipationEndDate time.Time

		err = rows.Scan(
			&resultPersistentGathering.Gathering.ID,
			&resultPersistentGathering.Gathering.OwnerPID,
			&resultPersistentGathering.Gathering.HostPID,
			&resultPersistentGathering.Gathering.MinimumParticipants,
			&resultPersistentGathering.Gathering.MaximumParticipants,
			&resultPersistentGathering.Gathering.ParticipationPolicy,
			&resultPersistentGathering.Gathering.PolicyArgument,
			&resultPersistentGathering.Gathering.Flags,
			&resultPersistentGathering.Gathering.State,
			&resultPersistentGathering.Gathering.Description,
			&resultPersistentGathering.CommunityType,
			&resultPersistentGathering.Password,
			pqextended.Array(&resultAttribs),
			&resultPersistentGathering.ApplicationBuffer,
			&resultParticipationStartDate,
			&resultParticipationEndDate,
			&resultPersistentGathering.MatchmakeSessionCount,
			&resultPersistentGathering.ParticipationCount,
		)
		if err != nil {
			common_globals.Logger.Critical(err.Error())
			continue
		}

		attributesSlice := make([]types.UInt32, len(resultAttribs))
		for i, value := range resultAttribs {
			attributesSlice[i] = types.NewUInt32(value)
		}
		resultPersistentGathering.Attribs = attributesSlice

		resultPersistentGathering.ParticipationStartDate = resultPersistentGathering.ParticipationStartDate.FromTimestamp(resultParticipationStartDate)
		resultPersistentGathering.ParticipationEndDate = resultPersistentGathering.ParticipationEndDate.FromTimestamp(resultParticipationEndDate)

		persistentGatherings = append(persistentGatherings, resultPersistentGathering)
	}

	rows.Close()

	return persistentGatherings, nil
}
