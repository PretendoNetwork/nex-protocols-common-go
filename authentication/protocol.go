package authentication

import (
	"github.com/PretendoNetwork/nex-go"
	_ "github.com/PretendoNetwork/nex-protocols-go"
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
}

func (commonAuthenticationProtocol *CommonAuthenticationProtocol) SetSecureStationURL(stationURL *nex.StationURL) {
	commonAuthenticationProtocol.secureStationURL = stationURL
}

func (commonAuthenticationProtocol *CommonAuthenticationProtocol) SetBuildName(buildName string) {
	commonAuthenticationProtocol.buildName = buildName
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
