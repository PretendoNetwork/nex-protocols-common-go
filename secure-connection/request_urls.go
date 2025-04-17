package secureconnection

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func (commonProtocol *CommonProtocol) requestURLs(err error, packet nex.PacketInterface, callID uint32, cidTarget types.UInt32, pidTarget types.PID) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	// TODO - Is this correct?
	requestedConnection := endpoint.FindConnectionByID(uint32(cidTarget))
	if requestedConnection == nil {
		requestedConnection = endpoint.FindConnectionByPID(uint64(pidTarget))
	}

	if requestedConnection == nil {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval := types.NewBool(true)
	retval.WriteTo(rmcResponseStream)

	plstURLs := requestedConnection.StationURLs
	plstURLs.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodRequestURLs
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterRequestURLs != nil {
		go commonProtocol.OnAfterRequestURLs(packet, cidTarget, pidTarget)
	}

	return rmcResponse, nil
}
