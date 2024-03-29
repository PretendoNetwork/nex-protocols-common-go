package datastore

import (
	"fmt"
	"time"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func prepareGetObject(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStorePrepareGetParam) uint32 {
	if commonDataStoreProtocol.getObjectInfoByDataIDHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.S3Presigner == nil {
		common_globals.Logger.Warning("S3Presigner not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.Unknown
	}

	client := packet.Sender()

	bucket := commonDataStoreProtocol.s3Bucket
	key := fmt.Sprintf("%s/%d.bin", commonDataStoreProtocol.s3DataKeyBase, param.DataID)

	objectInfo, errCode := commonDataStoreProtocol.getObjectInfoByDataIDHandler(param.DataID)
	if errCode != 0 {
		return errCode
	}

	errCode = commonDataStoreProtocol.VerifyObjectPermission(objectInfo.OwnerID, client.PID(), objectInfo.Permission)
	if errCode != 0 {
		return errCode
	}

	url, err := commonDataStoreProtocol.S3Presigner.GetObject(bucket, key, time.Minute*15)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.OperationNotAllowed
	}

	requestHeaders, errCode := commonDataStoreProtocol.s3GetRequestHeadersHandler()
	if errCode != 0 {
		return errCode
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

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodPrepareGetObject, rmcResponseBody)

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
