package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func (commonProtocol *CommonProtocol) getPasswordInfos(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	if len(dataIDs) > int(datastore_constants.BatchProcessingCapacity) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pPasswordInfos, pResults, errCode := database.GetObjectPasswords(manager, connection.PID(), dataIDs)
	if errCode != nil {
		return nil, errCode
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pPasswordInfos.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetPasswordInfos
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
