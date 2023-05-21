package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	"github.com/PretendoNetwork/plogger-go"
)

var commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol
var logger = plogger.NewLogger()

type CommonMatchmakeExtensionProtocol struct {
	*matchmake_extension.MatchmakeExtensionProtocol
	server *nex.Server

	CleanupSearchMatchmakeSessionHandler    func(matchmakeSession match_making.MatchmakeSession) match_making.MatchmakeSession
}

// NewCommonMatchmakeExtensionProtocol returns a new CommonMatchmakeExtensionProtocol
func NewCommonMatchmakeExtensionProtocol(server *nex.Server) *CommonMatchmakeExtensionProtocol {
	MatchmakeExtensionProtocol := matchmake_extension.NewMatchmakeExtensionProtocol(server)
	commonMatchmakeExtensionProtocol = &CommonMatchmakeExtensionProtocol{MatchmakeExtensionProtocol: MatchmakeExtensionProtocol, server: server}

	MatchmakeExtensionProtocol.AutoMatchmake_Postpone(AutoMatchmake_Postpone)

	return commonMatchmakeExtensionProtocol
}
