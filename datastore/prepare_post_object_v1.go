package datastore

import (
	"fmt"
	"math"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) preparePostObjectV1(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStorePreparePostParamV1) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager

	if manager.S3 == nil {
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "S3 config not set")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// TODO - Add rollback for when error occurs

	// * Hack to get this to work, just map the V1 data to the modern struct
	// TODO - Add a dedicated function for V1 inserts, for stuff like checking if we hit the uint32 limit?
	newParam := datastore_types.NewDataStorePreparePostParam()
	newParam.Size = param.Size
	newParam.Name = param.Name
	newParam.DataType = param.DataType
	newParam.MetaBinary = param.MetaBinary
	newParam.Permission = param.Permission
	newParam.DelPermission = param.DelPermission
	newParam.Flag = param.Flag
	newParam.Period = param.Period
	newParam.ReferDataID = param.ReferDataID
	newParam.Tags = param.Tags
	newParam.RatingInitParams = param.RatingInitParams

	notUseFileServer := (newParam.Flag & types.UInt32(datastore_constants.DataFlagNotUseFileServer)) != 0
	if notUseFileServer {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "PreparePostObjectV1 cannot be used with DataFlagNotUseFileServer")
	}

	dataID, errCode := database.InsertObjectByPreparePostParam(manager, connection.PID(), newParam)
	if errCode != nil {
		common_globals.Logger.Errorf("Error on object init: %s", errCode.Error())
		return nil, errCode
	}

	// * This should never happen, but bail right away if it does
	// TODO - Check if we hit the limit BEFORE inserting?
	if dataID > math.MaxUint32 {
		// * OverCapacity is definitely not the correct error here, but it sounds nice
		return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
	}

	// TODO - Should this be moved inside InsertObjectByPreparePostParam?
	errCode = database.InsertObjectRatingSettings(manager, dataID, newParam.RatingInitParams)
	if errCode != nil {
		common_globals.Logger.Errorf("Error on rating init: %s", errCode.Error())
		return nil, errCode
	}

	// * Format "DataID_Version", where "Version" always starts at 1
	key := fmt.Sprintf("%020d_%010d.bin", dataID, 1)
	postData, err := manager.S3.PresignPost(key, time.Minute*15)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "Failed to sign post request")
	}

	pReqPostInfo := datastore_types.NewDataStoreReqPostInfoV1()

	pReqPostInfo.DataID = types.NewUInt32(uint32(dataID))
	pReqPostInfo.URL = types.NewString(postData.URL.String())
	pReqPostInfo.RequestHeaders = types.NewList[datastore_types.DataStoreKeyValue]()
	pReqPostInfo.FormFields = types.NewList[datastore_types.DataStoreKeyValue]()
	pReqPostInfo.RootCACert = types.NewBuffer(postData.RootCACert)

	for key, value := range postData.RequestHeaders {
		header := datastore_types.NewDataStoreKeyValue()
		header.Key = types.NewString(key)
		header.Value = types.NewString(value)

		pReqPostInfo.RequestHeaders = append(pReqPostInfo.RequestHeaders, header)
	}

	for key, value := range postData.FormData {
		field := datastore_types.NewDataStoreKeyValue()
		field.Key = types.NewString(key)
		field.Value = types.NewString(value)

		pReqPostInfo.FormFields = append(pReqPostInfo.FormFields, field)
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pReqPostInfo.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPreparePostObjectV1
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
