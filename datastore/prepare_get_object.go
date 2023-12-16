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
	if commonProtocol.GetObjectInfoByDataID == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonProtocol.S3Presigner == nil {
		common_globals.Logger.Warning("S3Presigner not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.Unknown
	}

	server := commonProtocol.server
	client := packet.Sender()

	bucket := commonProtocol.S3Bucket
	key := fmt.Sprintf("%s/%d.bin", commonProtocol.s3DataKeyBase, param.DataID)

	objectInfo, errCode := commonProtocol.GetObjectInfoByDataID(param.DataID)
	if errCode != 0 {
		return nil, errCode
	}

	errCode = commonProtocol.VerifyObjectPermission(objectInfo.OwnerID, client.PID(), objectInfo.Permission)
	if errCode != 0 {
		return nil, errCode
	}

	url, err := commonProtocol.S3Presigner.GetObject(bucket, key, time.Minute*15)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.OperationNotAllowed
	}

	requestHeaders, errCode := commonProtocol.S3GetRequestHeaders()
	if errCode != 0 {
		return nil, errCode
	}

	pReqGetInfo := datastore_types.NewDataStoreReqGetInfo()

	pReqGetInfo.URL = url.String()
	pReqGetInfo.RequestHeaders = requestHeaders
	pReqGetInfo.Size = objectInfo.Size
	pReqGetInfo.RootCACert = commonProtocol.RootCACert
	pReqGetInfo.DataID = param.DataID

	rmcResponseStream := nex.NewStreamOut(commonProtocol.server)

	rmcResponseStream.WriteStructure(pReqGetInfo)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPrepareGetObject
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
