package secureconnection

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	_ "github.com/PretendoNetwork/nex-protocols-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"
)

type CommonProtocol struct {
	endpoint                    nex.EndpointInterface
	protocol                    secure_connection.Interface
	CreateReportDBRecord        func(pid types.PID, reportID types.UInt32, reportData types.QBuffer) error
	ValidateLoginData           func(pid types.PID, loginData types.DataHolder) *nex.Error
	allowInsecureRegisterMethod bool
	OnAfterRegister             func(packet nex.PacketInterface, vecMyURLs types.List[types.StationURL])
	OnAfterRequestURLs          func(packet nex.PacketInterface, cidTarget types.UInt32, pidTarget types.PID)
	OnAfterRegisterEx           func(packet nex.PacketInterface, vecMyURLs types.List[types.StationURL], hCustomData types.DataHolder)
	OnAfterReplaceURL           func(packet nex.PacketInterface, target types.StationURL, url types.StationURL)
	OnAfterSendReport           func(packet nex.PacketInterface, reportID types.UInt32, reportData types.QBuffer)
}

// DisableInsecureRegister disables the insecure Register method
func (commonProtocol *CommonProtocol) DisableInsecureRegister() {
	commonProtocol.allowInsecureRegisterMethod = false
}

// EnableInsecureRegister enables the insecure Register method. Do not enable this on NEX 1 games
func (commonProtocol *CommonProtocol) EnableInsecureRegister() {
	common_globals.Logger.Warning("Insecure Register has been enabled. This MUST NOT be enabled on NEX 1 games or with EnableInsecureLogin at the same time except on friends. Use with caution!")
	commonProtocol.allowInsecureRegisterMethod = true
}

// SetPretendoValidation configures the protocol to use Pretendo validation
func (commonProtocol *CommonProtocol) SetPretendoValidation(aesKey []byte) {
	commonProtocol.ValidateLoginData = func(pid types.PID, loginData types.DataHolder) *nex.Error {
		return common_globals.ValidatePretendoLoginData(pid, loginData, aesKey)
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol secure_connection.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerRegister(commonProtocol.register)
	protocol.SetHandlerRequestURLs(commonProtocol.requestURLs)
	protocol.SetHandlerRegisterEx(commonProtocol.registerEx)
	protocol.SetHandlerReplaceURL(commonProtocol.replaceURL)
	protocol.SetHandlerSendReport(commonProtocol.sendReport)

	return commonProtocol
}
