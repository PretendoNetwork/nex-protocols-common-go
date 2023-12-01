package datastore

import (
	"fmt"
	"time"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func preparePostObject(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStorePreparePostParam) (*nex.RMCMessage, uint32) {
	if commonDataStoreProtocol.InitializeObjectByPreparePostParam == nil {
		common_globals.Logger.Warning("InitializeObjectByPreparePostParam not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.InitializeObjectRatingWithSlot == nil {
		common_globals.Logger.Warning("InitializeObjectRatingWithSlot not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.S3Presigner == nil {
		common_globals.Logger.Warning("S3Presigner not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	client := packet.Sender().(*nex.PRUDPClient)

	// TODO - Need to verify what param.PersistenceInitParam.DeleteLastObject really means. It's often set to true even when it wouldn't make sense
	dataID, errCode := commonDataStoreProtocol.InitializeObjectByPreparePostParam(client.PID().LegacyValue(), param)
	if errCode != 0 {
		common_globals.Logger.Errorf("Error code %d on object init", errCode)
		return nil, errCode
	}

	// TODO - Should this be moved to InitializeObjectByPreparePostParam?
	for _, ratingInitParamWithSlot := range param.RatingInitParams {
		errCode = commonDataStoreProtocol.InitializeObjectRatingWithSlot(dataID, ratingInitParamWithSlot)
		if errCode != 0 {
			common_globals.Logger.Errorf("Error code %d on rating init", errCode)
			return nil, errCode
		}
	}

	bucket := commonDataStoreProtocol.S3Bucket
	key := fmt.Sprintf("%s/%d.bin", commonDataStoreProtocol.s3DataKeyBase, dataID)

	URL, formData, err := commonDataStoreProtocol.S3Presigner.PostObject(bucket, key, time.Minute*15)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.OperationNotAllowed
	}

	requestHeaders, errCode := commonDataStoreProtocol.S3PostRequestHeaders()
	if errCode != 0 {
		return nil, errCode
	}

	pReqPostInfo := datastore_types.NewDataStoreReqPostInfo()

	pReqPostInfo.DataID = dataID
	pReqPostInfo.URL = URL.String()
	pReqPostInfo.RequestHeaders = requestHeaders
	pReqPostInfo.FormFields = make([]*datastore_types.DataStoreKeyValue, 0, len(formData))
	pReqPostInfo.RootCACert = commonDataStoreProtocol.RootCACert

	for key, value := range formData {
		field := datastore_types.NewDataStoreKeyValue()
		field.Key = key
		field.Value = value

		pReqPostInfo.FormFields = append(pReqPostInfo.FormFields, field)
	}

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	rmcResponseStream.WriteStructure(pReqPostInfo)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPreparePostObject
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
