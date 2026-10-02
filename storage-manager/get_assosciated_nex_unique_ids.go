package storage_manager

import (
	"github.com/PretendoNetwork/nex-go/v2"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/storage-manager/database"
	storagemanager "github.com/PretendoNetwork/nex-protocols-go/v2/storage-manager"
)

/* ACCURACY: This is a *guessed* protocol, we don't actually have captures or know what it means yet. */
func (commonProtocol *CommonProtocol) getAssociatedNexUniqueIDs(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	ids, nexErr := database.GetUniqueIDsForPID(commonProtocol.manager, connection.PID())
	if nexErr != nil {
		commonglobals.Logger.Error(nexErr.Error())
		return nil, nexErr
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	ids.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = storagemanager.ProtocolID
	rmcResponse.MethodID = storagemanager.MethodUnk3
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
