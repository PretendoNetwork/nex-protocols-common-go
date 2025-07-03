package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
)

func (commonProtocol *CommonProtocol) getPasswordInfo(err error, packet nex.PacketInterface, callID uint32, dataID types.UInt64) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * Reuse the multiple version of this call
	dataIDs := types.NewList[types.UInt64]()
	dataIDs = append(dataIDs, dataID)

	passwords, results, errCode := database.GetObjectPasswords(manager, connection.PID(), dataIDs)
	if errCode != nil {
		return nil, errCode
	}

	pPasswordInfo := passwords[0]
	result := results[0]

	if result.IsError() {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pPasswordInfo.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetPasswordInfo
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
