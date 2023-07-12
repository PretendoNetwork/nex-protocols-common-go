package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	"github.com/PretendoNetwork/plogger-go"
)

var commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol
var logger = plogger.NewLogger()

type CommonMatchmakeExtensionProtocol struct {
	*matchmake_extension.MatchmakeExtensionProtocol
	server *nex.Server

	cleanupSearchMatchmakeSessionHandler func(matchmakeSession *match_making_types.MatchmakeSession)
}

// CleanupSearchMatchmakeSession sets the CleanupSearchMatchmakeSession handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) CleanupSearchMatchmakeSession(handler func(matchmakeSession *match_making_types.MatchmakeSession)) {
	commonMatchmakeExtensionProtocol.cleanupSearchMatchmakeSessionHandler = handler
}

// NewCommonMatchmakeExtensionProtocol returns a new CommonMatchmakeExtensionProtocol
func NewCommonMatchmakeExtensionProtocol(server *nex.Server) *CommonMatchmakeExtensionProtocol {
	MatchmakeExtensionProtocol := matchmake_extension.NewMatchmakeExtensionProtocol(server)
	commonMatchmakeExtensionProtocol = &CommonMatchmakeExtensionProtocol{MatchmakeExtensionProtocol: MatchmakeExtensionProtocol, server: server}

	MatchmakeExtensionProtocol.AutoMatchmake_Postpone(autoMatchmake_Postpone)
	MatchmakeExtensionProtocol.AutoMatchmakeWithSearchCriteria_Postpone(autoMatchmakeWithSearchCriteria_Postpone)
	MatchmakeExtensionProtocol.OpenParticipation(openParticipation)

	return commonMatchmakeExtensionProtocol
}
