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
	server           *nex.Server
	secureStationURL *nex.StationURL
	buildName        string
}

func (commonTicketGrantingProtocol *CommonTicketGrantingProtocol) SetSecureStationURL(stationURL *nex.StationURL) {
	commonTicketGrantingProtocol.secureStationURL = stationURL
}

func (commonTicketGrantingProtocol *CommonTicketGrantingProtocol) SetBuildName(buildName string) {
	commonTicketGrantingProtocol.buildName = buildName
}

// NewCommonTicketGrantingProtocol returns a new CommonTicketGrantingProtocol
func NewCommonTicketGrantingProtocol(server *nex.Server) *CommonTicketGrantingProtocol {
	ticketGrantingProtocol := ticket_granting.NewTicketGrantingProtocol(server)
	commonTicketGrantingProtocol = &CommonTicketGrantingProtocol{TicketGrantingProtocol: ticketGrantingProtocol, server: server}

	commonTicketGrantingProtocol.Login(login)
	commonTicketGrantingProtocol.LoginEx(loginEx)
	commonTicketGrantingProtocol.RequestTicket(requestTicket)

	return commonTicketGrantingProtocol
}
