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

func (commonProtocol *CommonProtocol) changeMeta(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreChangeMetaParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	var metaInfo datastore_types.DataStoreMetaInfo
	var updatePassword types.UInt64
	var errCode *nex.Error

	// * If using a PersistenceTarget, ignore the DataID
	if param.PersistenceTarget.OwnerID != 0 {
		metaInfo, updatePassword, errCode = database.GetChangeMetaObjectInfoByPersistenceTarget(manager, param.PersistenceTarget)
	} else if param.DataID != types.UInt64(datastore_constants.InvalidDataID) {
		metaInfo, updatePassword, errCode = database.GetChangeMetaObjectInfoByDataID(manager, param.DataID)
	} else {
		// * If both the PersistenceTarget and DataID are not set, bail
		errCode = nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if errCode != nil {
		return nil, errCode
	}

	errCode = manager.VerifyObjectUpdatePermission(connection.PID(), metaInfo, updatePassword, param.UpdatePassword)
	if errCode != nil {
		return nil, errCode
	}

	// * If the object is pending or rejected, only the owner can interact with it
	if metaInfo.OwnerID != connection.PID() && (metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) || metaInfo.Status == types.UInt8(datastore_constants.DataStatusRejected)) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
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
		return nil, nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareAccessPermission && !metaInfo.Permission.Equals(param.CompareParam.Permission) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareUpdatePermission && !metaInfo.DelPermission.Equals(param.CompareParam.DelPermission) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if comparePeriod && !metaInfo.Period.Equals(param.CompareParam.Period) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareMetaBinary && !metaInfo.MetaBinary.Equals(param.CompareParam.MetaBinary) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareTags && !metaInfo.Tags.Equals(param.CompareParam.Tags) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareDataType && !metaInfo.DataType.Equals(param.CompareParam.DataType) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareStatus && !metaInfo.Status.Equals(param.CompareParam.Status) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	errCode = database.UpdateObjectMetadata(manager, metaInfo, param)
	if errCode != nil {
		return nil, errCode
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodChangeMeta
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterChangeMeta != nil {
		go commonProtocol.OnAfterChangeMeta(packet, param)
	}

	return rmcResponse, nil
}
