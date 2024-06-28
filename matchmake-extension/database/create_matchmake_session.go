package database

import (
	"crypto/rand"
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// CreateMatchmakeSession creates a new MatchmakeSession on the database. No participants are added
func CreateMatchmakeSession(db *sql.DB, connection *nex.PRUDPConnection, matchmakeSession *match_making_types.MatchmakeSession) *nex.Error {
	startedTime, nexError := match_making_database.RegisterGathering(db, connection.PID(), matchmakeSession.Gathering, "MatchmakeSession")
	if nexError != nil {
		return nexError
	}

	attribs := make([]uint32, matchmakeSession.Attributes.Length())
	for i, value := range matchmakeSession.Attributes.Slice() {
		attribs[i] = value.Value
	}

	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	matchmakeParam := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	srVariant := types.NewVariant()
	srVariant.TypeID.Value = 3
	srVariant.Type = types.NewPrimitiveBool(true)
	matchmakeSession.MatchmakeParam.Params.Set(types.NewString("@SR"), srVariant)
	girVariant := types.NewVariant()
	girVariant.TypeID.Value = 1
	girVariant.Type = types.NewPrimitiveS64(3)
	matchmakeSession.MatchmakeParam.Params.Set(types.NewString("@GIR"), srVariant)

	matchmakeSession.MatchmakeParam.WriteTo(matchmakeParam)

	matchmakeSession.StartedTime = startedTime
	matchmakeSession.SessionKey.Value = make([]byte, 32)
	rand.Read(matchmakeSession.SessionKey.Value)

	_, err := db.Exec(`INSERT INTO matchmaking.matchmake_sessions (
		id,
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
		$11,
		$12,
		$13,
		$14,
		$15
	)`,
		matchmakeSession.Gathering.ID.Value,
		matchmakeSession.GameMode.Value,
		pqextended.Array(attribs),
		matchmakeSession.OpenParticipation.Value,
		matchmakeSession.MatchmakeSystemType.Value,
		matchmakeSession.ApplicationBuffer.Value,
		matchmakeSession.ProgressScore.Value,
		matchmakeSession.SessionKey.Value,
		matchmakeSession.Option.Value,
		matchmakeParam.Bytes(),
		matchmakeSession.UserPassword.Value,
		matchmakeSession.ReferGID.Value,
		matchmakeSession.UserPasswordEnabled.Value,
		matchmakeSession.SystemPasswordEnabled.Value,
		matchmakeSession.CodeWord.Value,
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
