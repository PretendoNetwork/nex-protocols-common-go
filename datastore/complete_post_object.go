package datastore

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func completePostObject(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStoreCompletePostParam) uint32 {
	if commonDataStoreProtocol.minIOClient == nil {
		common_globals.Logger.Warning("MinIOClient not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.getObjectInfoByDataIDHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.getObjectOwnerByDataIDHandler == nil {
		common_globals.Logger.Warning("GetObjectOwnerByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.getObjectSizeByDataIDHandler == nil {
		common_globals.Logger.Warning("GetObjectSizeByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.updateObjectUploadCompletedByDataIDHandler == nil {
		common_globals.Logger.Warning("UpdateObjectUploadCompletedByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.deleteObjectByDataIDHandler == nil {
		common_globals.Logger.Warning("DeleteObjectByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	client := packet.Sender()

	// * If GetObjectInfoByDataID returns data then that means
	// * the object has already been marked as uploaded. So do
	// * nothing
	objectInfo, _ := commonDataStoreProtocol.getObjectInfoByDataIDHandler(param.DataID)
	if objectInfo != nil {
		return nex.Errors.DataStore.PermissionDenied
	}

	// * Only allow an objects owner to make this request
	ownerPID, errCode := commonDataStoreProtocol.getObjectOwnerByDataIDHandler(param.DataID)
	if errCode != 0 {
		return errCode
	}

	if ownerPID != client.PID() {
		return nex.Errors.DataStore.PermissionDenied
	}

	bucket := commonDataStoreProtocol.s3Bucket
	key := fmt.Sprintf("%s/%d.bin", commonDataStoreProtocol.s3DataKeyBase, param.DataID)

	if param.IsSuccess {
		objectSizeS3, err := commonDataStoreProtocol.S3ObjectSize(bucket, key)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nex.Errors.DataStore.NotFound
		}

		objectSizeDB, errCode := commonDataStoreProtocol.getObjectSizeByDataIDHandler(param.DataID)
		if errCode != 0 {
			return errCode
		}

		if objectSizeS3 != uint64(objectSizeDB) {
			common_globals.Logger.Errorf("Object with DataID %d did not upload correctly! Mismatched sizes", param.DataID)
			// TODO - Is this a good error?
			return nex.Errors.DataStore.Unknown
		}

		errCode = commonDataStoreProtocol.updateObjectUploadCompletedByDataIDHandler(param.DataID, true)
		if errCode != 0 {
			return errCode
		}
	} else {
		errCode := commonDataStoreProtocol.deleteObjectByDataIDHandler(param.DataID)
		if errCode != 0 {
			return errCode
		}
	}

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodCompletePostObject, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if commonDataStoreProtocol.server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(packet.Destination())
	responsePacket.SetDestination(packet.Source())
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
