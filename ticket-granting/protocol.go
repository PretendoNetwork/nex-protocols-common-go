package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go"
	_ "github.com/PretendoNetwork/nex-protocols-go"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	server                     nex.ServerInterface
	protocol                   ticket_granting.Interface
	SecureStationURL           *nex.StationURL
	SpecialProtocols           []uint8
	StationURLSpecialProtocols *nex.StationURL
	BuildName                  string
	allowInsecureLoginMethod   bool
}

func (commonProtocol *CommonProtocol) DisableInsecureLogin() {
	commonProtocol.allowInsecureLoginMethod = false
}

func (commonProtocol *CommonProtocol) EnableInsecureLogin() {
	common_globals.Logger.Warning("INSECURE LOGIN HAS BEEN ENABLED. THIS ALLOWS THE USE OF CUSTOM CLIENTS TO BYPASS THE ACCOUNT SERVER AND CONNECT DIRECTLY TO THIS GAME SERVER, EVADING BANS! USE WITH CAUTION!")
	commonProtocol.allowInsecureLoginMethod = true
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol ticket_granting.Interface) *CommonProtocol {
	protocol.SetHandlerLogin(login)
	protocol.SetHandlerLoginEx(loginEx)
	protocol.SetHandlerRequestTicket(requestTicket)

	commonProtocol = &CommonProtocol{
		server:                     protocol.Server(),
		protocol:                   protocol,
		SecureStationURL:           nex.NewStationURL("prudp:/"),
		SpecialProtocols:           make([]uint8, 0),
		StationURLSpecialProtocols: nex.NewStationURL(""),
	}

	commonProtocol.DisableInsecureLogin() // * Disable insecure login by default

	return commonProtocol
}
