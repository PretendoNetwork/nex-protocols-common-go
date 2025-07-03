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

func (commonProtocol *CommonProtocol) getRating(err error, packet nex.PacketInterface, callID uint32, target datastore_types.DataStoreRatingTarget, accessPassword types.UInt64) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if target.DataID == types.UInt64(datastore_constants.InvalidDataID) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if target.Slot >= types.UInt8(datastore_constants.NumRatingSlot) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	metaInfo, objectAccessPassword, errCode := database.GetAccessObjectInfoByDataID(manager, target.DataID)
	if errCode != nil {
		return nil, errCode
	}

	errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, objectAccessPassword, accessPassword)
	if errCode != nil {
		return nil, errCode
	}

	// * The owner of an object can always view their objects, but normal users cannot
	if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
		if metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) {
			return nil, nex.NewError(nex.ResultCodes.DataStore.UnderReviewing, "change_error")
		}

		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	pRating, errCode := database.GetObjectRatingByDataIDAndSlot(manager, target.DataID, target.Slot)
	if errCode != nil {
		return nil, errCode
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pRating.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetRating
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
