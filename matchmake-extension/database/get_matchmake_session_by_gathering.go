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
	var gameMode uint32
	var resultAttribs []uint32
	var openParticipation bool
	var matchmakeSystemType uint32
	var applicationBuffer []byte
	var progressScore uint8
	var sessionKey []byte
	var option uint32
	var resultMatchmakeParam []byte
	var userPassword string
	var referGID uint32
	var userPasswordEnabled bool
	var systemPasswordEnabled bool
	var codeWord string

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
		&gameMode,
		pqextended.Array(&resultAttribs),
		&openParticipation,
		&matchmakeSystemType,
		&applicationBuffer,
		&progressScore,
		&sessionKey,
		&option,
		&resultMatchmakeParam,
		&userPassword,
		&referGID,
		&userPasswordEnabled,
		&systemPasswordEnabled,
		&codeWord,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return match_making_types.NewMatchmakeSession(), nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return match_making_types.NewMatchmakeSession(), nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	resultMatchmakeSession := match_making_types.NewMatchmakeSession()

	resultMatchmakeSession.Gathering = gathering
	resultMatchmakeSession.ParticipationCount = types.NewUInt32(participationCount)
	resultMatchmakeSession.StartedTime = startedTime
	resultMatchmakeSession.GameMode = types.NewUInt32(gameMode)

	attributesSlice := make([]types.UInt32, len(resultAttribs))
	for i, value := range resultAttribs {
		attributesSlice[i] = types.NewUInt32(value)
	}
	resultMatchmakeSession.Attributes = attributesSlice

	resultMatchmakeSession.OpenParticipation = types.NewBool(openParticipation)
	resultMatchmakeSession.MatchmakeSystemType = types.NewUInt32(matchmakeSystemType)
	resultMatchmakeSession.ApplicationBuffer = applicationBuffer
	resultMatchmakeSession.ProgressScore = types.NewUInt8(progressScore)
	resultMatchmakeSession.SessionKey = sessionKey
	resultMatchmakeSession.Option = types.UInt32(option)

	matchmakeParamBytes := nex.NewByteStreamIn(resultMatchmakeParam, endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	resultMatchmakeSession.MatchmakeParam.ExtractFrom(matchmakeParamBytes)

	resultMatchmakeSession.UserPassword = types.NewString(userPassword)
	resultMatchmakeSession.ReferGID = types.NewUInt32(referGID)
	resultMatchmakeSession.UserPasswordEnabled = types.NewBool(userPasswordEnabled)
	resultMatchmakeSession.SystemPasswordEnabled = types.NewBool(systemPasswordEnabled)
	resultMatchmakeSession.CodeWord = types.String(codeWord)

	return resultMatchmakeSession, nil
}
