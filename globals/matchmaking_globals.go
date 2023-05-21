package common_globals
import (
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

type CommonMatchmakeSession struct {
	GameMatchmakeSession match_making.MatchmakeSession //used by the game, contains the current state of the MatchmakeSession
	SearchMatchmakeSession match_making.MatchmakeSession //used by the server when searching for matches, contains the state of the MatchmakeSession during the search process for easy compares
	ConnectionIDs []uint32 //players in the room, referenced by their connection IDs. This is used instead of the PID in order to ensure we're talking to the correct client (in case of e.g. multiple logins)
}

var Sessions = []CommonMatchmakeSession{}