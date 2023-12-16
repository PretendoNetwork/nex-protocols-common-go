package datastore

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func completePostObject(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStoreCompletePostParam) (*nex.RMCMessage, uint32) {
	if commonProtocol.minIOClient == nil {
		common_globals.Logger.Warning("MinIOClient not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonProtocol.GetObjectInfoByDataID == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonProtocol.GetObjectOwnerByDataID == nil {
		common_globals.Logger.Warning("GetObjectOwnerByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonProtocol.GetObjectSizeByDataID == nil {
		common_globals.Logger.Warning("GetObjectSizeByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonProtocol.UpdateObjectUploadCompletedByDataID == nil {
		common_globals.Logger.Warning("UpdateObjectUploadCompletedByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonProtocol.DeleteObjectByDataID == nil {
		common_globals.Logger.Warning("DeleteObjectByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	server := commonProtocol.server
	client := packet.Sender()

	// * If GetObjectInfoByDataID returns data then that means
	// * the object has already been marked as uploaded. So do
	// * nothing
	objectInfo, _ := commonProtocol.GetObjectInfoByDataID(param.DataID)
	if objectInfo != nil {
		return nil, nex.Errors.DataStore.PermissionDenied
	}

	// * Only allow an objects owner to make this request
	ownerPID, errCode := commonProtocol.GetObjectOwnerByDataID(param.DataID)
	if errCode != 0 {
		return nil, errCode
	}

	if ownerPID != client.PID().LegacyValue() {
		return nil, nex.Errors.DataStore.PermissionDenied
	}

	bucket := commonProtocol.S3Bucket
	key := fmt.Sprintf("%s/%d.bin", commonProtocol.s3DataKeyBase, param.DataID)

	if param.IsSuccess {
		objectSizeS3, err := commonProtocol.S3ObjectSize(bucket, key)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nil, nex.Errors.DataStore.NotFound
		}

		objectSizeDB, errCode := commonProtocol.GetObjectSizeByDataID(param.DataID)
		if errCode != 0 {
			return nil, errCode
		}

		if objectSizeS3 != uint64(objectSizeDB) {
			common_globals.Logger.Errorf("Object with DataID %d did not upload correctly! Mismatched sizes", param.DataID)
			// TODO - Is this a good error?
			return nil, nex.Errors.DataStore.Unknown
		}

		errCode = commonProtocol.UpdateObjectUploadCompletedByDataID(param.DataID, true)
		if errCode != 0 {
			return nil, errCode
		}
	} else {
		errCode := commonProtocol.DeleteObjectByDataID(param.DataID)
		if errCode != 0 {
			return nil, errCode
		}
	}

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodCompletePostObject
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
