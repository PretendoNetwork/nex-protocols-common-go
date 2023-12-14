package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go"
	_ "github.com/PretendoNetwork/nex-protocols-go"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

var commonTicketGrantingProtocol *CommonTicketGrantingProtocol

type CommonTicketGrantingProtocol struct {
	server                     nex.ServerInterface
	protocol                   ticket_granting.Interface
	SecureStationURL           *nex.StationURL
	SpecialProtocols           []uint8
	StationURLSpecialProtocols *nex.StationURL
	BuildName                  string
	allowInsecureLoginMethod   bool
}

func (commonTicketGrantingProtocol *CommonTicketGrantingProtocol) DisableInsecureLogin() {
	commonTicketGrantingProtocol.allowInsecureLoginMethod = false
}

func (commonTicketGrantingProtocol *CommonTicketGrantingProtocol) EnableInsecureLogin() {
	common_globals.Logger.Warning("INSECURE LOGIN HAS BEEN ENABLED. THIS ALLOWS THE USE OF CUSTOM CLIENTS TO BYPASS THE ACCOUNT SERVER AND CONNECT DIRECTLY TO THIS GAME SERVER, EVADING BANS! USE WITH CAUTION!")
	commonTicketGrantingProtocol.allowInsecureLoginMethod = true
}

// NewCommonTicketGrantingProtocol returns a new CommonTicketGrantingProtocol
func NewCommonTicketGrantingProtocol(protocol ticket_granting.Interface) *CommonTicketGrantingProtocol {
	protocol.SetHandlerLogin(login)
	protocol.SetHandlerLoginEx(loginEx)
	protocol.SetHandlerRequestTicket(requestTicket)

	commonTicketGrantingProtocol = &CommonTicketGrantingProtocol{
		server:                     protocol.Server(),
		protocol:                   protocol,
		SecureStationURL:           nex.NewStationURL("prudp:/"),
		SpecialProtocols:           make([]uint8, 0),
		StationURLSpecialProtocols: nex.NewStationURL("prudp:/"),
	}

	commonTicketGrantingProtocol.DisableInsecureLogin() // * Disable insecure login by default

	return commonTicketGrantingProtocol
}
