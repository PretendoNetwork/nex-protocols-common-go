package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	_ "github.com/PretendoNetwork/nex-protocols-go/v2"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

type CommonProtocol struct {
	protocol                   ticket_granting.Interface
	SecureStationURL           types.StationURL
	SpecialProtocols           []types.UInt8
	StationURLSpecialProtocols types.StationURL
	BuildName                  types.String
	allowInsecureLoginMethod   bool
	SessionKeyLength           int // TODO - Use server SessionKeyLength?
	SecureServerAccount        *nex.Account
	SourceKeyFromToken         func(sourceAccount *nex.Account, loginData types.DataHolder) ([]byte, *nex.Error)
	ValidateLoginData          func(pid types.PID, loginData types.DataHolder) *nex.Error
	OnAfterLogin               func(packet nex.PacketInterface, strUserName types.String)
	OnAfterLoginEx             func(packet nex.PacketInterface, strUserName types.String, oExtraData types.DataHolder)
	OnAfterRequestTicket       func(packet nex.PacketInterface, idSource types.PID, idTarget types.PID)
}

// DisableInsecureLogin disables the insecure Login method
func (commonProtocol *CommonProtocol) DisableInsecureLogin() {
	commonProtocol.allowInsecureLoginMethod = false
}

// EnableInsecureLogin enables the insecure Login method. Do not enable this on any servers outside friends and NEX 1 games
func (commonProtocol *CommonProtocol) EnableInsecureLogin() {
	common_globals.Logger.Warning("Insecure Login has been enabled. This MUST NOT be enabled on any servers outside friends or NEX 1 games or with EnableInsecureRegister at the same time. Use with caution!")
	commonProtocol.allowInsecureLoginMethod = true
}

// SetPretendoValidation configures the protocol to use Pretendo validation
func (commonProtocol *CommonProtocol) SetPretendoValidation(aesKey []byte) {
	commonProtocol.ValidateLoginData = func(pid types.PID, loginData types.DataHolder) *nex.Error {
		return common_globals.ValidatePretendoLoginData(pid, loginData, aesKey)
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol ticket_granting.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		protocol:                   protocol,
		SecureStationURL:           types.NewStationURL("prudp:/"),
		SpecialProtocols:           make([]types.UInt8, 0),
		StationURLSpecialProtocols: types.NewStationURL(""),
		BuildName:                  types.NewString(""),
		allowInsecureLoginMethod:   false,
		SessionKeyLength:           32,
	}

	protocol.SetHandlerLogin(commonProtocol.login)
	protocol.SetHandlerLoginEx(commonProtocol.loginEx)
	protocol.SetHandlerRequestTicket(commonProtocol.requestTicket)

	commonProtocol.DisableInsecureLogin() // * Disable insecure login by default

	return commonProtocol
}
