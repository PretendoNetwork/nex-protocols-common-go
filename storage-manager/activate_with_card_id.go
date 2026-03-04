package storage_manager

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/storage-manager/database"
	storagemanager "github.com/PretendoNetwork/nex-protocols-go/v2/storage-manager"
)

func (commonProtocol *CommonProtocol) setHandlerActivateWithCardID(err error, packet nex.PacketInterface, callID uint32, slot types.UInt8, cardID types.UInt64) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// Conversion to int64 is ok here because we generate the card IDs serverside and ensure they are within range
	uniqueID, firstTime, nexErr := database.GetUniqueId(commonProtocol.manager, slot, int64(cardID), connection.PID())
	if nexErr != nil {
		commonglobals.Logger.Error(nexErr.Error())
		return nil, nexErr
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	commonglobals.Logger.Infof("Unique ID: %v First time: %v Card ID: %v", uniqueID, firstTime, cardID)
	uniqueID.WriteTo(rmcResponseStream)
	firstTime.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = storagemanager.ProtocolID
	rmcResponse.MethodID = storagemanager.MethodActivateWithCardID
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
