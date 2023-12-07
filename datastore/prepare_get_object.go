package datastore

import (
	"fmt"
	"time"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func prepareGetObject(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStorePrepareGetParam) (*nex.RMCMessage, uint32) {
	if commonDataStoreProtocol.getObjectInfoByDataIDHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.S3Presigner == nil {
		common_globals.Logger.Warning("S3Presigner not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.Unknown
	}

	client := packet.Sender()

	bucket := commonDataStoreProtocol.s3Bucket
	key := fmt.Sprintf("%s/%d.bin", commonDataStoreProtocol.s3DataKeyBase, param.DataID)

	objectInfo, errCode := commonDataStoreProtocol.getObjectInfoByDataIDHandler(param.DataID)
	if errCode != 0 {
		return nil, errCode
	}

	errCode = commonDataStoreProtocol.VerifyObjectPermission(objectInfo.OwnerID, client.PID(), objectInfo.Permission)
	if errCode != 0 {
		return nil, errCode
	}

	url, err := commonDataStoreProtocol.S3Presigner.GetObject(bucket, key, time.Minute*15)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.OperationNotAllowed
	}

	requestHeaders, errCode := commonDataStoreProtocol.s3GetRequestHeadersHandler()
	if errCode != 0 {
		return nil, errCode
	}

	pReqGetInfo := datastore_types.NewDataStoreReqGetInfo()

	pReqGetInfo.URL = url.String()
	pReqGetInfo.RequestHeaders = requestHeaders
	pReqGetInfo.Size = objectInfo.Size
	pReqGetInfo.RootCACert = commonDataStoreProtocol.rootCACert
	pReqGetInfo.DataID = param.DataID

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	rmcResponseStream.WriteStructure(pReqGetInfo)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPrepareGetObject
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
