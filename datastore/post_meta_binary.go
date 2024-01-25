package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func postMetaBinary(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStorePreparePostParam) (*nex.RMCMessage, uint32) {
	// * This method looks to function identically to DataStore::PreparePostObject,
	// * except the only difference being it doesn't return an S3 upload URL. This
	// * needs to be verified though, as there are other methods in the family such
	// * as DataStore::PostMetaBinaryWithDataID which make less sense in this context,
	// * unless those are just used to *update* a meta binary? Or maybe the DataID in
	// * those methods is a pre-allocated DataID from the server? Needs more testing

	if commonProtocol.InitializeObjectByPreparePostParam == nil {
		common_globals.Logger.Warning("InitializeObjectByPreparePostParam not defined")
		return nil, nex.ResultCodes.Core.NotImplemented
	}

	if commonProtocol.InitializeObjectRatingWithSlot == nil {
		common_globals.Logger.Warning("InitializeObjectRatingWithSlot not defined")
		return nil, nex.ResultCodes.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.DataStore.Unknown
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	// TODO - Need to verify what param.PersistenceInitParam.DeleteLastObject really means. It's often set to true even when it wouldn't make sense
	dataID, errCode := commonProtocol.InitializeObjectByPreparePostParam(connection.PID(), param)
	if errCode != 0 {
		common_globals.Logger.Errorf("Error code %d on object init", errCode)
		return nil, errCode
	}

	// TODO - Should this be moved to InitializeObjectByPreparePostParam?
	param.RatingInitParams.Each(func(_ int, ratingInitParamWithSlot *datastore_types.DataStoreRatingInitParamWithSlot) bool {
		errCode = commonProtocol.InitializeObjectRatingWithSlot(dataID, ratingInitParamWithSlot)
		if errCode != 0 {
			common_globals.Logger.Errorf("Error code %d on rating init", errCode)
			return true
		}

		return false
	})

	if errCode != 0 {
		return nil, errCode
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	rmcResponseStream.WritePrimitiveUInt64LE(dataID)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPostMetaBinary
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
