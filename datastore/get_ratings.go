package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) getRatings(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64], accessPassword types.UInt64) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * This might actually be BatchProcessingCapacity.
	// * Using BatchProcessingCapacityPostObject for now
	// * to match RateObjects/RateObjectsWithPosting.
	// TODO - If we see a real client use more than 16, update this
	if len(dataIDs) > int(datastore_constants.BatchProcessingCapacityPostObject) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	pRatings := types.NewList[types.List[datastore_types.DataStoreRatingInfoWithSlot]]()
	pResults := types.NewList[types.QResult]()
	invalidRatingInfoWithSlot := types.NewList[datastore_types.DataStoreRatingInfoWithSlot]() // * Quick hack to get a zeroed struct

	// TODO - Optimize this, this can make dozens of database calls
	for _, dataID := range dataIDs {
		if dataID == types.UInt64(datastore_constants.InvalidDataID) {
			pRatings = append(pRatings, invalidRatingInfoWithSlot)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		metaInfo, objectAccessPassword, errCode := database.GetAccessObjectInfoByDataID(manager, dataID)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfoWithSlot)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, objectAccessPassword, accessPassword)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfoWithSlot)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
			if metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) {
				pRatings = append(pRatings, invalidRatingInfoWithSlot)
				pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.UnderReviewing))
				continue
			}

			pRatings = append(pRatings, invalidRatingInfoWithSlot)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			continue
		}

		ratings, errCode := database.GetObjectRatingsByDataID(manager, dataID)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfoWithSlot)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		pRatings = append(pRatings, ratings)
		pResults = append(pResults, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown))
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pRatings.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetRatings
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
