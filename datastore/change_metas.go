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

func (commonProtocol *CommonProtocol) changeMetas(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64], params types.List[datastore_types.DataStoreChangeMetaParam], transactional types.Bool) (*nex.RMCMessage, *nex.Error) {
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

		var metaInfo datastore_types.DataStoreMetaInfo
		var updatePassword types.UInt64
		var errCode *nex.Error

		// * If using a PersistenceTarget, ignore the DataID
		if param.PersistenceTarget.OwnerID != 0 {
			metaInfo, updatePassword, errCode = database.GetUpdateObjectInfoByPersistenceTarget(manager, param.PersistenceTarget)
		} else if dataID != types.UInt64(datastore_constants.InvalidDataID) {
			metaInfo, updatePassword, errCode = database.GetUpdateObjectInfoByDataID(manager, dataID)
		} else if param.DataID != types.UInt64(datastore_constants.InvalidDataID) {
			// * Unsure if this is accurate actually, but why not check it. Who cares?
			// * Normally this should be 0, but if we hit this point might as well try
			metaInfo, updatePassword, errCode = database.GetUpdateObjectInfoByDataID(manager, param.DataID)
		} else {
			// * If both the PersistenceTarget and DataID are not set, bail
			errCode = nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
		}

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

		compareName := (param.CompareParam.ComparisonFlag & types.UInt32(datastore_constants.ComparisonFlagName)) != 0
		compareAccessPermission := (param.CompareParam.ComparisonFlag & types.UInt32(datastore_constants.ComparisonFlagAccessPermission)) != 0
		compareUpdatePermission := (param.CompareParam.ComparisonFlag & types.UInt32(datastore_constants.ComparisonFlagUpdatePermission)) != 0
		comparePeriod := (param.CompareParam.ComparisonFlag & types.UInt32(datastore_constants.ComparisonFlagPeriod)) != 0
		compareMetaBinary := (param.CompareParam.ComparisonFlag & types.UInt32(datastore_constants.ComparisonFlagMetaBinary)) != 0
		compareTags := (param.CompareParam.ComparisonFlag & types.UInt32(datastore_constants.ComparisonFlagTags)) != 0
		compareDataType := (param.CompareParam.ComparisonFlag & types.UInt32(datastore_constants.ComparisonFlagDataType)) != 0
		compareStatus := (param.CompareParam.ComparisonFlag & types.UInt32(datastore_constants.ComparisonFlagStatus)) != 0

		if param.CompareParam.ComparisonFlag == types.UInt32(datastore_constants.ComparisonFlagAll) {
			compareName = true
			compareAccessPermission = true
			compareUpdatePermission = true
			comparePeriod = true
			compareMetaBinary = true
			compareTags = true
			compareDataType = true
			compareStatus = true
		}

		if compareName && !metaInfo.Name.Equals(param.CompareParam.Name) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.ValueNotEqual))
			continue
		}

		if compareAccessPermission && !metaInfo.Permission.Equals(param.CompareParam.Permission) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.ValueNotEqual))
			continue
		}

		if compareUpdatePermission && !metaInfo.DelPermission.Equals(param.CompareParam.DelPermission) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.ValueNotEqual))
			continue
		}

		if comparePeriod && !metaInfo.Period.Equals(param.CompareParam.Period) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.ValueNotEqual))
			continue
		}

		if compareMetaBinary && !metaInfo.MetaBinary.Equals(param.CompareParam.MetaBinary) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.ValueNotEqual))
			continue
		}

		if compareTags && !metaInfo.Tags.Equals(param.CompareParam.Tags) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.ValueNotEqual))
			continue
		}

		if compareDataType && !metaInfo.DataType.Equals(param.CompareParam.DataType) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.ValueNotEqual))
			continue
		}

		if compareStatus && !metaInfo.Status.Equals(param.CompareParam.Status) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.ValueNotEqual))
			continue
		}

		errCode = database.UpdateObjectMetadata(manager, metaInfo, param)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		pResults = append(pResults, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown))
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodChangeMetas
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
