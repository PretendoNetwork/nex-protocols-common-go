package datastore

import (
	"fmt"

	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
)

func completePostObjects(err error, packet nex.PacketInterface, callID uint32, dataIDs []uint64) uint32 {
	if commonDataStoreProtocol.minIOClient == nil {
		common_globals.Logger.Warning("MinIOClient not defined")
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

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	client := packet.Sender().(*nex.PRUDPClient)

	for _, dataID := range dataIDs {
		bucket := commonDataStoreProtocol.s3Bucket
		key := fmt.Sprintf("%s/%d.bin", commonDataStoreProtocol.s3DataKeyBase, dataID)

		objectSizeS3, err := commonDataStoreProtocol.S3ObjectSize(bucket, key)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nex.Errors.DataStore.NotFound
		}

		objectSizeDB, errCode := commonDataStoreProtocol.getObjectSizeByDataIDHandler(dataID)
		if errCode != 0 {
			return errCode
		}

		if objectSizeS3 != uint64(objectSizeDB) {
			common_globals.Logger.Errorf("Object with DataID %d did not upload correctly! Mismatched sizes", dataID)
			// TODO - Is this a good error?
			return nex.Errors.DataStore.Unknown
		}

		errCode = commonDataStoreProtocol.updateObjectUploadCompletedByDataIDHandler(dataID, true)
		if errCode != 0 {
			return errCode
		}
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodCompletePostObjects
	rmcResponse.CallID = callID

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if commonDataStoreProtocol.server.PRUDPVersion == 0 {
		responsePacket, _ = nex.NewPRUDPPacketV0(client, nil)
	} else {
		responsePacket, _ = nex.NewPRUDPPacketV1(client, nil)
	}

	responsePacket.SetType(nex.DataPacket)
	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)
	responsePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	responsePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	responsePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	responsePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	responsePacket.SetPayload(rmcResponseBytes)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
