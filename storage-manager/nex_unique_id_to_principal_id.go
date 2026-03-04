package storage_manager

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/storage-manager/database"
	storagemanager "github.com/PretendoNetwork/nex-protocols-go/v2/storage-manager"
)

func (commonProtocol *CommonProtocol) nexUniqueIdToPrincipalId(err error, packet nex.PacketInterface, callID uint32, uniqueID types.UInt32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pid, nexErr := database.GetPidForUniqueId(commonProtocol.manager, uniqueID)
	if nexErr != nil {
		commonglobals.Logger.Error(nexErr.Error())
		return nil, nexErr
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pid.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = storagemanager.ProtocolID
	rmcResponse.MethodID = storagemanager.MethodNexUniqueIDToPrincipalID
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
