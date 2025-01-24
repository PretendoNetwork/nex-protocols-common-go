package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// GetMatchmakeSessionByGathering gets a matchmake session with the given gathering data
func GetMatchmakeSessionByGathering(manager *common_globals.MatchmakingManager, endpoint *nex.PRUDPEndPoint, gathering match_making_types.Gathering, participationCount uint32, startedTime types.DateTime) (match_making_types.MatchmakeSession, *nex.Error) {
	resultMatchmakeSession := match_making_types.NewMatchmakeSession()
	var resultAttribs []uint32
	var resultMatchmakeParam []byte

	err := manager.Database.QueryRow(`SELECT
		game_mode,
		attribs,
		open_participation,
		matchmake_system_type,
		application_buffer,
		progress_score,
		session_key,
		option_zero,
		matchmake_param,
		user_password,
		refer_gid,
		user_password_enabled,
		system_password_enabled,
		codeword
		FROM matchmaking.matchmake_sessions WHERE id=$1`,
		uint32(gathering.ID),
	).Scan(
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
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return match_making_types.NewMatchmakeSession(), nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return match_making_types.NewMatchmakeSession(), nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	resultMatchmakeSession.Gathering = gathering
	resultMatchmakeSession.ParticipationCount = types.NewUInt32(participationCount)
	resultMatchmakeSession.StartedTime = startedTime

	attributesSlice := make([]types.UInt32, len(resultAttribs))
	for i, value := range resultAttribs {
		attributesSlice[i] = types.NewUInt32(value)
	}
	resultMatchmakeSession.Attributes = attributesSlice

	matchmakeParamBytes := nex.NewByteStreamIn(resultMatchmakeParam, endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	resultMatchmakeSession.MatchmakeParam.ExtractFrom(matchmakeParamBytes)

	return resultMatchmakeSession, nil
}
