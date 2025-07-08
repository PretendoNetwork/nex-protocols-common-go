package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// GetPlayingSession returns the playing sessions of the given PIDs
func GetPlayingSession(manager *common_globals.MatchmakingManager, listPID types.List[types.PID]) (types.List[match_making_types.PlayingSession], *nex.Error) {
	playingSessions := make([]match_making_types.PlayingSession, 0, 1000) // * Allocate for a capacity of up to MAX_MATCHMAKE_SESSION_BY_PARTICIPANT entries
	for _, pid := range listPID {
		if len(playingSessions) >= 1000 { // * MAX_MATCHMAKE_SESSION_BY_PARTICIPANT
			break
		}

		// TODO - Handle user blocks when implemented
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
			ms.option_zero,
			ms.matchmake_param,
			ms.refer_gid,
			ms.user_password_enabled,
			ms.system_password_enabled,
			ms.codeword
		FROM matchmaking.gatherings AS g
		INNER JOIN matchmaking.matchmake_sessions AS ms ON ms.id = g.id
		WHERE
		g.registered=true AND
		g.type='MatchmakeSession' AND
		$1=ANY(g.participants)`, pid)
		if err != nil {
			common_globals.Logger.Critical(err.Error())
			continue
		}

		for rows.Next() {
			if len(playingSessions) >= 1000 { // * MAX_MATCHMAKE_SESSION_BY_PARTICIPANT
				break
			}

			playingSession := match_making_types.NewPlayingSession()
			resultMatchmakeSession := match_making_types.NewMatchmakeSession()
			var resultMatchmakeParam []byte

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
				&resultMatchmakeSession.StartedTime,
				&resultMatchmakeSession.GameMode,
				&resultMatchmakeSession.Attributes,
				&resultMatchmakeSession.OpenParticipation,
				&resultMatchmakeSession.MatchmakeSystemType,
				&resultMatchmakeSession.ApplicationBuffer,
				&resultMatchmakeSession.ProgressScore,
				&resultMatchmakeSession.Option,
				&resultMatchmakeParam,
				&resultMatchmakeSession.ReferGID,
				&resultMatchmakeSession.UserPasswordEnabled,
				&resultMatchmakeSession.SystemPasswordEnabled,
				&resultMatchmakeSession.CodeWord,
			)
			if err != nil {
				common_globals.Logger.Critical(err.Error())
				continue
			}

			matchmakeParamBytes := nex.NewByteStreamIn(resultMatchmakeParam, manager.Endpoint.LibraryVersions(), manager.Endpoint.ByteStreamSettings())
			resultMatchmakeSession.MatchmakeParam.ExtractFrom(matchmakeParamBytes)

			playingSession.PrincipalID = pid
			playingSession.Gathering.Object = resultMatchmakeSession

			playingSessions = append(playingSessions, playingSession)
		}

		rows.Close()
	}

	return playingSessions, nil
}
