package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go"
	_ "github.com/PretendoNetwork/nex-protocols-go"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"
	"github.com/PretendoNetwork/plogger-go"
)

var commonTicketGrantingProtocol *CommonTicketGrantingProtocol
var logger = plogger.NewLogger()

type CommonTicketGrantingProtocol struct {
	*ticket_granting.TicketGrantingProtocol
	server                   *nex.Server
	secureStationURL         *nex.StationURL
	buildName                string
	allowInsecureLoginMethod bool
}

func (commonTicketGrantingProtocol *CommonTicketGrantingProtocol) SetSecureStationURL(stationURL *nex.StationURL) {
	commonTicketGrantingProtocol.secureStationURL = stationURL
}

func (commonTicketGrantingProtocol *CommonTicketGrantingProtocol) SetBuildName(buildName string) {
	commonTicketGrantingProtocol.buildName = buildName
}

func (commonTicketGrantingProtocol *CommonTicketGrantingProtocol) DisableInsecureLogin() {
	commonTicketGrantingProtocol.allowInsecureLoginMethod = false
}

func (commonTicketGrantingProtocol *CommonTicketGrantingProtocol) EnableInsecureLogin() {
	logger.Warning("INSECURE LOGIN HAS BEEN ENABLED. THIS ALLOWS THE USE OF CUSTOM CLIENTS TO BYPASS THE ACCOUNT SERVER AND CONNECT DIRECTLY TO THIS GAME SERVER, EVADING BANS! USE WITH CAUTION!")
	commonTicketGrantingProtocol.allowInsecureLoginMethod = true
}

// NewCommonTicketGrantingProtocol returns a new CommonTicketGrantingProtocol
func NewCommonTicketGrantingProtocol(server *nex.Server) *CommonTicketGrantingProtocol {
	ticketGrantingProtocol := ticket_granting.NewTicketGrantingProtocol(server)
	commonTicketGrantingProtocol = &CommonTicketGrantingProtocol{
		TicketGrantingProtocol: ticketGrantingProtocol,
		server:                 server,
	}

	commonTicketGrantingProtocol.DisableInsecureLogin() // * Disable insecure login by default
	commonTicketGrantingProtocol.Login(login)
	commonTicketGrantingProtocol.LoginEx(loginEx)
	commonTicketGrantingProtocol.RequestTicket(requestTicket)

	return commonTicketGrantingProtocol
}
