package database

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// GetMatchmakeSessionByID gets a matchmake session with the given gathering ID and the system password
func GetMatchmakeSessionByID(db *sql.DB, endpoint *nex.PRUDPEndPoint, gatheringID uint32) (*match_making_types.MatchmakeSession, string, *nex.Error) {
	resultMatchmakeSession := match_making_types.NewMatchmakeSession()
	var ownerPID uint64
	var hostPID uint64
	var startedTime time.Time
	var resultAttribs []uint32
	var resultMatchmakeParam []byte
	var systemPassword string

	// * For simplicity, we will only compare the values that exist on a MatchmakeSessionSearchCriteria
	err := db.QueryRow(`SELECT
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
		&resultMatchmakeSession.Gathering.ID.Value,
		&ownerPID,
		&hostPID,
		&resultMatchmakeSession.Gathering.MinimumParticipants.Value,
		&resultMatchmakeSession.Gathering.MaximumParticipants.Value,
		&resultMatchmakeSession.Gathering.ParticipationPolicy.Value,
		&resultMatchmakeSession.Gathering.PolicyArgument.Value,
		&resultMatchmakeSession.Gathering.Flags.Value,
		&resultMatchmakeSession.Gathering.State.Value,
		&resultMatchmakeSession.Gathering.Description.Value,
		&resultMatchmakeSession.ParticipationCount.Value,
		&startedTime,
		&resultMatchmakeSession.GameMode.Value,
		pqextended.Array(&resultAttribs),
		&resultMatchmakeSession.OpenParticipation.Value,
		&resultMatchmakeSession.MatchmakeSystemType.Value,
		&resultMatchmakeSession.ApplicationBuffer.Value,
		&resultMatchmakeSession.ProgressScore.Value,
		&resultMatchmakeSession.SessionKey.Value,
		&resultMatchmakeSession.Option.Value,
		&resultMatchmakeParam,
		&resultMatchmakeSession.UserPassword.Value,
		&resultMatchmakeSession.ReferGID.Value,
		&resultMatchmakeSession.UserPasswordEnabled.Value,
		&resultMatchmakeSession.SystemPasswordEnabled.Value,
		&resultMatchmakeSession.CodeWord.Value,
		&systemPassword,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return nil, "", nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	resultMatchmakeSession.OwnerPID = types.NewPID(ownerPID)
	resultMatchmakeSession.HostPID = types.NewPID(hostPID)
	resultMatchmakeSession.StartedTime = resultMatchmakeSession.StartedTime.FromTimestamp(startedTime)

	attributesSlice := make([]*types.PrimitiveU32, len(resultAttribs))
	for i, value := range resultAttribs {
		attributesSlice[i] = types.NewPrimitiveU32(value)
	}
	resultMatchmakeSession.Attributes.SetFromData(attributesSlice)

	matchmakeParamBytes := nex.NewByteStreamIn(resultMatchmakeParam, endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	resultMatchmakeSession.MatchmakeParam.ExtractFrom(matchmakeParamBytes)

	return resultMatchmakeSession, systemPassword, nil
}
