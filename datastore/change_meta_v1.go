package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) changeMetaV1(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreChangeMetaParamV1) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if param.DataID == types.UInt64(datastore_constants.InvalidDataID) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	// * V1 has no persistence target
	metaInfo, updatePassword, errCode := database.GetUpdateObjectInfoByDataID(manager, types.UInt64(param.DataID))
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

	// * If the object is pending or rejected, only the owner can interact with it
	if metaInfo.OwnerID != connection.PID() && (metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) || metaInfo.Status == types.UInt8(datastore_constants.DataStatusRejected)) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	errCode = database.UpdateObjectMetadataV1(manager, metaInfo, param)
	if errCode != nil {
		return nil, errCode
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodChangeMetaV1
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
