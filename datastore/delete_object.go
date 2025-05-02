package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) deleteObject(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreDeleteParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	metaInfo, updatePassword, errCode := database.GetUpdateObjectInfoByDataID(manager, param.DataID)
	if errCode != nil {
		return nil, errCode
	}

	// TODO - Move this to VerifyObjectUpdatePermission?
	// * Objects in the DataID range 900,000-999,999 are special
	if metaInfo.DataID < 1000000 {
		// * Unsure if this is the correct error, but it feels right
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	errCode = manager.VerifyObjectUpdatePermission(connection.PID(), metaInfo, updatePassword, param.UpdatePassword)
	if errCode != nil {
		return nil, errCode
	}

	errCode = database.DeleteObject(manager, param.DataID)
	if errCode != nil {
		return nil, errCode
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodDeleteObject
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
