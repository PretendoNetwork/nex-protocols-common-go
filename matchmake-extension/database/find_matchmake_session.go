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

// FindMatchmakeSession finds a matchmake session with the given search matchmake session
func FindMatchmakeSession(db *sql.DB, connection *nex.PRUDPConnection, searchMatchmakeSession *match_making_types.MatchmakeSession) (*match_making_types.MatchmakeSession, *nex.Error) {
	attribs := make([]uint32, searchMatchmakeSession.Attributes.Length())
	for i, value := range searchMatchmakeSession.Attributes.Slice() {
		attribs[i] = value.Value
	}

	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	searchStatement := `SELECT
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
		ms.codeword
		FROM matchmaking.gatherings AS g
		INNER JOIN matchmaking.matchmake_sessions AS ms ON ms.id = g.id
		WHERE
		g.registered=true AND
		g.type='MatchmakeSession' AND
		ms.open_participation=true AND
		array_length(g.participants, 1) < g.max_participants AND
		g.max_participants=$1 AND
		g.min_participants=$2 AND
		ms.game_mode=$3 AND
		ms.attribs=$4 AND
		ms.matchmake_system_type=$5 AND
		ms.refer_gid=$6 AND
		ms.codeword=$7 AND (CASE WHEN g.participation_policy=98 THEN g.owner_pid=ANY($8) ELSE true END)`

	var friendList []uint32
	// * Prevent access to friend rooms if not implemented
	if common_globals.GetUserFriendPIDsHandler != nil {
		friendList = common_globals.GetUserFriendPIDsHandler(connection.PID().LegacyValue())
	}

	resultMatchmakeSession := match_making_types.NewMatchmakeSession()
	var ownerPID uint64
	var hostPID uint64
	var startedTime time.Time
	var resultAttribs []uint32
	var resultMatchmakeParam []byte

	// * For simplicity, we will only compare the values that exist on a MatchmakeSessionSearchCriteria
	err := db.QueryRow(searchStatement,
		searchMatchmakeSession.Gathering.MaximumParticipants.Value,
		searchMatchmakeSession.Gathering.MinimumParticipants.Value,
		searchMatchmakeSession.GameMode.Value,
		pqextended.Array(attribs),
		searchMatchmakeSession.MatchmakeSystemType.Value,
		searchMatchmakeSession.ReferGID.Value,
		searchMatchmakeSession.CodeWord.Value,
		pqextended.Array(friendList),
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
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
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

	return resultMatchmakeSession, nil
}
