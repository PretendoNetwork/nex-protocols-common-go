package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func getRelaySignatureKey(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	server := commonProtocol.server

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteInt32LE(0) // * Relay mode
	rmcResponseStream.WriteDateTime(nex.NewDateTime(0).Now())
	rmcResponseStream.WriteString("")  // * Relay server address. We don't have one, so for now this is empty.
	rmcResponseStream.WriteUInt16LE(0) // * Relay server port. We don't have one, so for now this is empty.
	rmcResponseStream.WriteInt32LE(0)  // * Relay address type
	rmcResponseStream.WriteUInt32LE(0) // * Game Server ID. I don't know if this is checked (it doesn't appear to be though).

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = nat_traversal.ProtocolID
	rmcResponse.MethodID = nat_traversal.MethodGetRelaySignatureKey
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
