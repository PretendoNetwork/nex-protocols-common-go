package match_making_ext

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
	"github.com/PretendoNetwork/plogger-go"
)

var commonMatchMakingExtProtocol *CommonMatchMakingExtProtocol
var logger = plogger.NewLogger()

type CommonMatchMakingExtProtocol struct {
	*match_making_ext.MatchMakingExtProtocol
	server *nex.Server
}

// NewCommonMatchmakeExtensionProtocol returns a new CommonMatchmakeExtensionProtocol
func NewCommonMatchMakingExtProtocol(server *nex.Server) *CommonMatchMakingExtProtocol {
	MatchMakingExtProtocol := match_making_ext.NewMatchMakingExtProtocol(server)
	commonMatchMakingExtProtocol = &CommonMatchMakingExtProtocol{MatchMakingExtProtocol: MatchMakingExtProtocol, server: server}

	MatchMakingExtProtocol.EndParticipation(EndParticipation)

	return commonMatchMakingExtProtocol
}
