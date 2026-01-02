package secureconnection

import (
	"github.com/PretendoNetwork/nex-go/v2"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func (commonProtocol *CommonProtocol) testConnectivity(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	// TODO - Implement
	// This is only to make games that use it not immediately disconnect upon call

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint()

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodTestConnectivity
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
