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

func (commonProtocol *CommonProtocol) getSpecificMeta(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreGetSpecificMetaParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if len(param.DataIDs) > int(datastore_constants.BatchProcessingCapacity) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	pMetaInfos := types.NewList[datastore_types.DataStoreSpecificMetaInfo]()

	// * Check this before hitting the DB
	for _, dataID := range param.DataIDs {
		if dataID == types.UInt64(datastore_constants.InvalidDataID) {
			return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
		}
	}

	for _, dataID := range param.DataIDs {
		metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByDataID(manager, dataID)
		if errCode != nil {
			return nil, errCode
		}

		errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, 0)
		if errCode != nil {
			return nil, errCode
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
			if metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) {
				return nil, nex.NewError(nex.ResultCodes.DataStore.UnderReviewing, "change_error")
			}

			return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
		}

		version, errCode := database.GetObjectLatestVersionNumber(manager, metaInfo.DataID)
		if errCode != nil {
			return nil, errCode
		}

		specificMetaInfo := datastore_types.NewDataStoreSpecificMetaInfo()
		specificMetaInfo.DataID = metaInfo.DataID
		specificMetaInfo.OwnerID = metaInfo.OwnerID
		specificMetaInfo.Size = metaInfo.Size
		specificMetaInfo.DataType = metaInfo.DataType
		specificMetaInfo.Version = types.UInt32(version)

		pMetaInfos = append(pMetaInfos, specificMetaInfo)
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pMetaInfos.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetSpecificMeta
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
