package common_globals

import (
	"github.com/PretendoNetwork/nex-go"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
)

type CommonMatchmakeSession struct {
	GameMatchmakeSession   *match_making_types.MatchmakeSession                 // * Used by the game, contains the current state of the MatchmakeSession
	SearchMatchmakeSession *match_making_types.MatchmakeSession                 // * Used by the server when searching for matches, contains the state of the MatchmakeSession during the search process for easy compares
	SearchCriteria         []*match_making_types.MatchmakeSessionSearchCriteria // * Used by the server when searching for matches, contains the list of MatchmakeSessionSearchCriteria
	ConnectionIDs          []uint32                                             // * Players in the room, referenced by their connection IDs. This is used instead of the PID in order to ensure we're talking to the correct client (in case of e.g. multiple logins)
}

var Sessions map[uint32]*CommonMatchmakeSession
var CurrentGatheringID = nex.NewCounter(0)
var CurrentMatchmakingCallID = nex.NewCounter(0)
