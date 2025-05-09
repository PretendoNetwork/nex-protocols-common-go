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

func (commonProtocol *CommonProtocol) changeMetasV1(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64], params types.List[datastore_types.DataStoreChangeMetaParamV1], transactional types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if len(dataIDs) != len(params) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if len(params) > int(datastore_constants.BatchProcessingCapacity) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	pResults := types.NewList[types.QResult]()

	// TODO - Support transactional. If transactional=true, changes are all-or-nothing
	// TODO - Refactor this, this can make hundreds of database calls in one go
	for i := 0; i < len(dataIDs); i++ {
		dataID := dataIDs[i]
		param := params[i]

		if dataID == types.UInt64(datastore_constants.InvalidDataID) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		// * V1 has no persistence target
		metaInfo, updatePassword, errCode := database.GetUpdateObjectInfoByDataID(manager, dataID)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// TODO - Move this to VerifyObjectUpdatePermission?
		// * Objects in the DataID range 900,000-999,999 are special
		if metaInfo.DataID < 1000000 {
			// * Unsure if this is the correct error, but it feels right
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.OperationNotAllowed))
			continue
		}

		errCode = manager.VerifyObjectUpdatePermission(connection.PID(), metaInfo, updatePassword, param.UpdatePassword)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * If the object is pending or rejected, only the owner can interact with it
		if metaInfo.OwnerID != connection.PID() && (metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) || metaInfo.Status == types.UInt8(datastore_constants.DataStatusRejected)) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			continue
		}

		errCode = database.UpdateObjectMetadataV1(manager, metaInfo, param)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodChangeMetasV1
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
