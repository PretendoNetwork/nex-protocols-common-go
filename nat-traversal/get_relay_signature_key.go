package nattraversal

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

func getRelaySignatureKey(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	relayMode := types.NewPrimitiveS32(0)        // * Relay mode? No idea what this means
	currentUTCTime := types.NewDateTime(0).Now() // Current time for the relay server, UTC
	address := types.NewString("")               // * Relay server address. We don't have one, so for now this is empty.
	port := types.NewPrimitiveU16(0)             // * Relay server port. We don't have one, so for now this is empty.
	relayAddressType := types.NewPrimitiveS32(0) // * Relay address type? No idea what this means
	gameServerID := types.NewPrimitiveU32(0)     // * Game Server ID. I don't know if this is checked (it doesn't appear to be though).

	rmcResponseStream := nex.NewByteStreamOut(server)

	relayMode.WriteTo(rmcResponseStream)
	currentUTCTime.WriteTo(rmcResponseStream)
	address.WriteTo(rmcResponseStream)
	port.WriteTo(rmcResponseStream)
	relayAddressType.WriteTo(rmcResponseStream)
	gameServerID.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = nat_traversal.ProtocolID
	rmcResponse.MethodID = nat_traversal.MethodGetRelaySignatureKey
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
