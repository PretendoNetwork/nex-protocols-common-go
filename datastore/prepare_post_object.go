package datastore

import (
	"fmt"
	"time"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func preparePostObject(err error, client *nex.Client, callID uint32, param *datastore_types.DataStorePreparePostParam) uint32 {
	if commonDataStoreProtocol.initializeObjectByPreparePostParamHandler == nil {
		common_globals.Logger.Warning("InitializeObjectByPreparePostParam not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.initializeObjectRatingWithSlotHandler == nil {
		common_globals.Logger.Warning("InitializeObjectRatingWithSlot not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.S3Presigner == nil {
		common_globals.Logger.Warning("S3Presigner not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

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

	bucket := commonDataStoreProtocol.s3Bucket
	key := fmt.Sprintf("%s/%d.bin", commonDataStoreProtocol.s3DataKeyBase, dataID)

	URL, formData, err := commonDataStoreProtocol.S3Presigner.PostObject(bucket, key, time.Minute*15)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.OperationNotAllowed
	}

	requestHeaders, errCode := commonDataStoreProtocol.s3PostRequestHeadersHandler()
	if errCode != 0 {
		return errCode
	}

	pReqPostInfo := datastore_types.NewDataStoreReqPostInfo()

	pReqPostInfo.DataID = dataID
	pReqPostInfo.URL = URL.String()
	pReqPostInfo.RequestHeaders = requestHeaders
	pReqPostInfo.FormFields = make([]*datastore_types.DataStoreKeyValue, 0, len(formData))
	pReqPostInfo.RootCACert = commonDataStoreProtocol.rootCACert

	for key, value := range formData {
		field := datastore_types.NewDataStoreKeyValue()
		field.Key = key
		field.Value = value

		pReqPostInfo.FormFields = append(pReqPostInfo.FormFields, field)
	}

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	rmcResponseStream.WriteStructure(pReqPostInfo)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodPreparePostObject, rmcResponseBody)

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
