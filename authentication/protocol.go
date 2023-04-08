package authentication

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-protocols-go/authentication"
	"github.com/PretendoNetwork/plogger-go"
)

var commonAuthenticationProtocol *CommonAuthenticationProtocol
var logger = plogger.NewLogger()

type CommonAuthenticationProtocol struct {
	*authentication.AuthenticationProtocol
	server                 *nex.Server
	secureStationURL       *nex.StationURL
	buildName              string
	passwordFromPIDHandler func(pid uint32) (string, uint32)
}

func (commonAuthenticationProtocol *CommonAuthenticationProtocol) SetSecureStationURL(stationURL *nex.StationURL) {
	commonAuthenticationProtocol.secureStationURL = stationURL
}

func (commonAuthenticationProtocol *CommonAuthenticationProtocol) SetBuildName(buildName string) {
	commonAuthenticationProtocol.buildName = buildName
}

func (commonAuthenticationProtocol *CommonAuthenticationProtocol) SetPasswordFromPIDFunction(handler func(pid uint32) (string, uint32)) {
	commonAuthenticationProtocol.passwordFromPIDHandler = handler
}

// NewCommonAuthenticationProtocol returns a new CommonAuthenticationProtocol
func NewCommonAuthenticationProtocol(server *nex.Server) *CommonAuthenticationProtocol {
	authenticationProtocol := authentication.NewAuthenticationProtocol(server)
	commonAuthenticationProtocol = &CommonAuthenticationProtocol{AuthenticationProtocol: authenticationProtocol, server: server}

	commonAuthenticationProtocol.Login(login)
	commonAuthenticationProtocol.LoginEx(loginEx)
	commonAuthenticationProtocol.RequestTicket(requestTicket)

	return commonAuthenticationProtocol
}
