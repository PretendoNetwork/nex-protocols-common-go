package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) searchObject(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreSearchParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	connection := packet.Sender()
	manager := commonProtocol.manager
	endpoint := connection.Endpoint()

	// * This is likely game-specific. Also developer note:
	// * Please keep in mind that no results is allowed. errCode
	// * should NEVER be DataStore::NotFound!
	// *
	// * DataStoreSearchParam contains a ResultRange to limit the
	// * returned results. TotalCount is the total matching objects
	// * in the database, whereas objects is the limited results
	objects, totalCount, errCode := database.GetObjectMetaInfosByDataStoreSearchParam(commonProtocol.manager, param, connection.PID())
	if errCode != nil {
		return nil, errCode
	}

	pSearchResult := datastore_types.NewDataStoreSearchResult()

	pSearchResult.Result = types.NewList[datastore_types.DataStoreMetaInfo]()

	for _, object := range objects {
		// TODO passwords???
		errCode = manager.VerifyObjectAccessPermission(*manager, connection.PID(), object, 0, 0)
		if errCode != nil {
			// * Since we don't error here, should we also
			// * "hide" these results by also decrementing
			// * totalCount?
			continue
		}

		object.FilterPropertiesByResultOption(param.ResultOption)

		pSearchResult.Result = append(pSearchResult.Result, object)
	}

	var totalCountType uint8

	// * Doing this here since the object
	// * the permissions checks in the
	// * previous loop will mutate the data
	// * returned from the database
	if totalCount == uint32(len(pSearchResult.Result)) {
		totalCountType = 0 // * Has no more data. All possible results were returned
	} else {
		totalCountType = 1 // * Has more data. Not all possible results were returned
	}

	// * Disables the TotalCount
	// *
	// * Only seen in struct revision 3 or
	// * NEX 4.0+
	if param.StructureVersion >= 3 || endpoint.LibraryVersions().DataStore.GreaterOrEqual("4.0.0") {
		if !param.TotalCountEnabled {
			totalCount = 0
			totalCountType = 3
		}
	}

	pSearchResult.TotalCount = types.NewUInt32(totalCount)
	pSearchResult.TotalCountType = types.NewUInt8(totalCountType)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pSearchResult.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodSearchObject
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
