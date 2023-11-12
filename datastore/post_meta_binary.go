package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func postMetaBinary(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStorePreparePostParam) uint32 {
	// * This method looks to function identically to DataStore::PreparePostObject,
	// * except the only difference being it doesn't return an S3 upload URL. This
	// * needs to be verified though, as there are other methods in the family such
	// * as DataStore::PostMetaBinaryWithDataID which make less sense in this context,
	// * unless those are just used to *update* a meta binary? Or maybe the DataID in
	// * those methods is a pre-allocated DataID from the server? Needs more testing

	if commonDataStoreProtocol.initializeObjectByPreparePostParamHandler == nil {
		common_globals.Logger.Warning("InitializeObjectByPreparePostParam not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.initializeObjectRatingWithSlotHandler == nil {
		common_globals.Logger.Warning("InitializeObjectRatingWithSlot not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	client := packet.Sender().(*nex.PRUDPClient)

	// TODO - Need to verify what param.PersistenceInitParam.DeleteLastObject really means. It's often set to true even when it wouldn't make sense
	dataID, errCode := commonDataStoreProtocol.initializeObjectByPreparePostParamHandler(client.PID(), param)
	if errCode != 0 {
		common_globals.Logger.Errorf("Error code %d on object init", errCode)
		return errCode
	}

	// TODO - Should this be moved to InitializeObjectByPreparePostParam?
	for _, ratingInitParamWithSlot := range param.RatingInitParams {
		errCode = commonDataStoreProtocol.initializeObjectRatingWithSlotHandler(dataID, ratingInitParamWithSlot)
		if errCode != 0 {
			common_globals.Logger.Errorf("Error code %d on rating init", errCode)
			return errCode
		}
	}

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	rmcResponseStream.WriteUInt64LE(uint64(dataID))

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPostMetaBinary
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
