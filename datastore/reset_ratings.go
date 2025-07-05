package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func (commonProtocol *CommonProtocol) resetRatings(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64], transactional types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * This IS BatchProcessingCapacity, unlike other rating methods
	if len(dataIDs) > int(datastore_constants.BatchProcessingCapacity) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	// TODO - Add rollback

	pResults := types.NewList[types.QResult]()

	for _, dataID := range dataIDs {
		if dataID == types.UInt64(datastore_constants.InvalidDataID) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		metaInfo, objectUpdatePassword, errCode := database.GetUpdateObjectInfoByDataID(manager, dataID)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		errCode = manager.VerifyObjectUpdatePermission(*manager, connection.PID(), metaInfo, objectUpdatePassword, 0)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
			if metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) {
				pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.UnderReviewing))
				continue
			}

			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			continue
		}

		errCode = database.DeleteObjectRatings(manager, dataID)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		pResults = append(pResults, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown))
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodResetRatings
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
