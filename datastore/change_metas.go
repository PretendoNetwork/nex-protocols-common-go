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

	if len(dataIDs) > datastore_constants.BatchProcessingCapacity || len(params) > datastore_constants.BatchProcessingCapacity {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
	}

	// * A request is only valid if it's either many DataIDs with one param,
	// * or no DataIDs with many params
	useDataIDList := len(dataIDs) != 0
	if (useDataIDList && len(params) != 1) || (!useDataIDList && len(params) == 0) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	targets := len(params)
	if useDataIDList {
		targets = len(dataIDs)
	}

	pResults := types.NewList[types.QResult]()

	// TODO - Support transactional. If transactional=true, changes are all-or-nothing
	// TODO - Refactor this, this can make hundreds of database calls in one go
	for i := 0; i < targets; i++ {
		var metaInfo datastore_types.DataStoreMetaInfo
		var updatePassword types.UInt64
		var errCode *nex.Error

		param := params[0]
		if !useDataIDList {
			param = params[i]
		}

		if useDataIDList {
			metaInfo, updatePassword, errCode = database.GetUpdateObjectInfoByDataID(manager.Database, dataIDs[i])
		} else if param.PersistenceTarget.PersistenceSlotID != types.UInt16(datastore_constants.InvalidPersistenceSlotID) {
			metaInfo, updatePassword, errCode = database.GetUpdateObjectInfoByPersistenceTarget(manager.Database, param.PersistenceTarget)
		} else {
			metaInfo, updatePassword, errCode = database.GetUpdateObjectInfoByDataID(manager.Database, param.DataID)
		}

		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// TODO - Move this to VerifyObjectUpdatePermission?
		// * Objects in the DataID range 900,000-999,999 belong to the system, and a
		// * normal client is not allowed to touch them
		// * https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
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
		if metaInfo.OwnerID != connection.PID() && (metaInfo.Status == datastore_constants.DataStatusPending || metaInfo.Status == datastore_constants.DataStatusRejected) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.UnderReviewing))
			continue
		}

		compareName := param.CompareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagName)
		compareAccessPermission := param.CompareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagAccessPermission)
		compareUpdatePermission := param.CompareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagUpdatePermission)
		comparePeriod := param.CompareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagPeriod)
		compareMetaBinary := param.CompareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagMetaBinary)
		compareTags := param.CompareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagTags)
		compareDataType := param.CompareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagDataType)
		compareStatus := param.CompareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagStatus)

		if param.CompareParam.ComparisonFlag == datastore_constants.ComparisonFlagAll {
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

		if compareStatus && metaInfo.Status != param.CompareParam.Status {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.ValueNotEqual))
			continue
		}

		errCode = database.UpdateObjectMetadata(manager.Database, metaInfo, param)
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
