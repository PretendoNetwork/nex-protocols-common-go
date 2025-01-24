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

// GetMatchmakeSessionByID gets a matchmake session with the given gathering ID and the system password
func GetMatchmakeSessionByID(manager *common_globals.MatchmakingManager, endpoint *nex.PRUDPEndPoint, gatheringID uint32) (match_making_types.MatchmakeSession, string, *nex.Error) {
	resultMatchmakeSession := match_making_types.NewMatchmakeSession()
	var startedTime time.Time
	var resultAttribs []uint32
	var resultMatchmakeParam []byte
	var systemPassword string

	// * For simplicity, we will only compare the values that exist on a MatchmakeSessionSearchCriteria
	err := manager.Database.QueryRow(`SELECT
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
		g.id=$1`,
		gatheringID,
	).Scan(
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
		&resultMatchmakeSession.Option,
		&resultMatchmakeParam,
		&resultMatchmakeSession.UserPassword,
		&resultMatchmakeSession.ReferGID,
		&resultMatchmakeSession.UserPasswordEnabled,
		&resultMatchmakeSession.SystemPasswordEnabled,
		&resultMatchmakeSession.CodeWord,
		&systemPassword,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return match_making_types.NewMatchmakeSession(), "", nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return match_making_types.NewMatchmakeSession(), "", nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	resultMatchmakeSession.StartedTime = resultMatchmakeSession.StartedTime.FromTimestamp(startedTime)

	attributesSlice := make([]types.UInt32, len(resultAttribs))
	for i, value := range resultAttribs {
		attributesSlice[i] = types.NewUInt32(value)
	}
	resultMatchmakeSession.Attributes = attributesSlice

	matchmakeParamBytes := nex.NewByteStreamIn(resultMatchmakeParam, endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	resultMatchmakeSession.MatchmakeParam.ExtractFrom(matchmakeParamBytes)

	return resultMatchmakeSession, systemPassword, nil
}
