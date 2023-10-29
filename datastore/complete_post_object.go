package datastore

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func completePostObject(err error, client *nex.Client, callID uint32, param *datastore_types.DataStoreCompletePostParam) uint32 {
	if commonDataStoreProtocol.s3ObjectSizeHandler == nil {
		common_globals.Logger.Warning("S3ObjectSize not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.getObjectSizeDataIDHandler == nil {
		common_globals.Logger.Warning("GetObjectSizeDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.updateObjectUploadCompletedByDataIDHandler == nil {
		common_globals.Logger.Warning("UpdateObjectUploadCompletedByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.deleteObjectByDataIDHandler == nil {
		common_globals.Logger.Warning("DeleteObjectByDataIDHandler not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	if param.IsSuccess {
		bucket := commonDataStoreProtocol.s3Bucket
		key := fmt.Sprintf("%d.bin", param.DataID)

		objectSizeS3, err := commonDataStoreProtocol.s3ObjectSizeHandler(bucket, key)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nex.Errors.DataStore.NotFound
		}

		objectSizeDB, errCode := commonDataStoreProtocol.getObjectSizeDataIDHandler(param.DataID)
		if errCode != 0 {
			return errCode
		}

		if objectSizeS3 != uint64(objectSizeDB) {
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

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
