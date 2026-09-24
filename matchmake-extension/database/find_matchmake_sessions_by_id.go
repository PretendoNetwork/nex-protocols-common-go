package database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// FindMatchmakeSessionByID finds matchmake sessions with the given list of Gathering IDs
func FindMatchmakeSessionsByID(manager *common_globals.MatchmakingManager, endpoint *nex.PRUDPEndPoint, lstGid types.List[types.UInt32]) ([]match_making_types.MatchmakeSession, *nex.Error) {
	matchmakeSessions := make([]match_making_types.MatchmakeSession, 0)

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
		array_length(g.participants, 1),
		g.started_time,
		ms.game_mode,
		ms.attribs,
		ms.open_participation,
		ms.matchmake_system_type,
		ms.application_buffer,
		ms.progress_score,
		ms.session_key,
		ms.option_zero,
		ms.matchmake_param,
		ms.user_password,
		ms.refer_gid,
		ms.user_password_enabled,
		ms.system_password_enabled,
		ms.codeword,
		ms.system_password
		FROM matchmaking.gatherings AS g
		INNER JOIN matchmaking.matchmake_sessions AS ms ON ms.id = g.id
		WHERE
		g.registered=true AND
		g.type='MatchmakeSession' AND
		ms.open_participation=true AND
		array_length(g.participants, 1) < g.max_participants AND
		g.host_pid <> 0 AND
		ms.user_password_enabled=false AND
		ms.system_password_enabled=false AND
		g.id=ANY($1)`,
		pqextended.Array(lstGid),
	)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}

	for rows.Next() {
		resultMatchmakeSession := match_making_types.NewMatchmakeSession()
		var startedTime time.Time
		var resultAttribs []uint32
		var resultMatchmakeParam []byte
		var systemPassword string

		err = rows.Scan(
			&resultMatchmakeSession.Gathering.ID,
			&resultMatchmakeSession.Gathering.OwnerPID,
			&resultMatchmakeSession.Gathering.HostPID,
			&resultMatchmakeSession.Gathering.MinimumParticipants,
			&resultMatchmakeSession.Gathering.MaximumParticipants,
			&resultMatchmakeSession.Gathering.ParticipationPolicy,
			&resultMatchmakeSession.Gathering.PolicyArgument,
			&resultMatchmakeSession.Gathering.Flags,
			&resultMatchmakeSession.Gathering.State,
			&resultMatchmakeSession.Gathering.Description,
			&resultMatchmakeSession.ParticipationCount,
			&startedTime,
			&resultMatchmakeSession.GameMode,
			pqextended.Array(&resultAttribs),
			&resultMatchmakeSession.OpenParticipation,
			&resultMatchmakeSession.MatchmakeSystemType,
			&resultMatchmakeSession.ApplicationBuffer,
			&resultMatchmakeSession.ProgressScore,
			&resultMatchmakeSession.SessionKey,
			&resultMatchmakeSession.Option0,
			&resultMatchmakeParam,
			&resultMatchmakeSession.UserPassword,
			&resultMatchmakeSession.ReferGID,
			&resultMatchmakeSession.UserPasswordEnabled,
			&resultMatchmakeSession.SystemPasswordEnabled,
			&resultMatchmakeSession.CodeWord,
			&systemPassword,
		)

		resultMatchmakeSession.StartedTime = resultMatchmakeSession.StartedTime.FromTimestamp(startedTime)

		attributesSlice := make([]types.UInt32, len(resultAttribs))
		for i, value := range resultAttribs {
			attributesSlice[i] = types.NewUInt32(value)
		}
		resultMatchmakeSession.Attributes = attributesSlice

		matchmakeParamBytes := nex.NewByteStreamIn(resultMatchmakeParam, endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
		resultMatchmakeSession.MatchmakeParam.ExtractFrom(matchmakeParamBytes)

		matchmakeSessions = append(matchmakeSessions, resultMatchmakeSession)
	}

	rows.Close()

	return matchmakeSessions, nil
}
