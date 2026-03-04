package storage_manager

import (
	"math/rand/v2"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	storagemanager "github.com/PretendoNetwork/nex-protocols-go/v2/storage-manager"
)

/* We don't actually need to store the card ID or anything here, just return a randomly generated one.
 * The rest of the database code is explicitly set up to allow overlap in card IDs between users - this is a required
 * behavior for accuracy, but we can use it to our advantage to simplify implementation too.
 */
func (commonProtocol *CommonProtocol) acquireCardId(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	id := rand.Int64() // 63 bits to fit in PostgreSQL bigint
	retval := types.NewUInt64(uint64(id))

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)
	commonglobals.Logger.Infof("Card ID: %v", retval)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = storagemanager.ProtocolID
	rmcResponse.MethodID = storagemanager.MethodAcquireCardID
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
