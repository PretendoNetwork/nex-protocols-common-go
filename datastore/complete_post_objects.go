package datastore

import (
	"fmt"

	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
)

func completePostObjects(err error, client *nex.Client, callID uint32, dataIDs []uint64) uint32 {
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

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	for _, dataID := range dataIDs {
		bucket := commonDataStoreProtocol.s3Bucket
		key := fmt.Sprintf("%d.bin", dataID)

		objectSizeS3, err := commonDataStoreProtocol.s3ObjectSizeHandler(bucket, key)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nex.Errors.DataStore.NotFound
		}

		objectSizeDB, errCode := commonDataStoreProtocol.getObjectSizeDataIDHandler(dataID)
		if errCode != 0 {
			return errCode
		}

		if objectSizeS3 != uint64(objectSizeDB) {
			// TODO - Is this a good error?
			return nex.Errors.DataStore.Unknown
		}

		errCode = commonDataStoreProtocol.updateObjectUploadCompletedByDataIDHandler(dataID, true)
		if errCode != 0 {
			return errCode
		}
	}

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodCompletePostObjects, nil)

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