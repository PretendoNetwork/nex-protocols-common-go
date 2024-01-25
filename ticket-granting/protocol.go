package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	_ "github.com/PretendoNetwork/nex-protocols-go"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	protocol                   ticket_granting.Interface
	SecureStationURL           *types.StationURL
	SpecialProtocols           []*types.PrimitiveU8
	StationURLSpecialProtocols *types.StationURL
	BuildName                  *types.String
	allowInsecureLoginMethod   bool
	SessionKeyLength           int
	SecureServerAccount        *nex.Account
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
		protocol:                   protocol,
		SecureStationURL:           types.NewStationURL("prudp:/"),
		SpecialProtocols:           make([]*types.PrimitiveU8, 0),
		StationURLSpecialProtocols: types.NewStationURL(""),
		BuildName:                  types.NewString(""),
		allowInsecureLoginMethod:   false,
		SessionKeyLength:           32,
	}

	commonProtocol.DisableInsecureLogin() // * Disable insecure login by default

	return commonProtocol
}
