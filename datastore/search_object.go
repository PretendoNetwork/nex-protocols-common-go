package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	"github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) searchObject(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreSearchParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * prior to revision 3 the total count was always enabled.
	// * NEX 4 resets the revision count back to 0 so we have to
	// * account for that too
	totalCountEnabled := true
	if param.StructureVersion >= 3 || endpoint.LibraryVersions().DataStore.GreaterOrEqual("4.0.0") {
		totalCountEnabled = bool(param.TotalCountEnabled)
	}

	// * Using the whole encoded param as the cache key.
	// * Unsure if this is correct but works for me
	paramStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	param.WriteTo(paramStream)

	cacheKey := paramStream.Bytes()

	var rmcResponseBody []byte
	var errCode *nex.Error

	if param.UseCache {
		rmcResponseBody, errCode = database.GetCachedSearchObjectResponse(manager.Database, connection.PID(), cacheKey)
		if errCode != nil {
			return nil, errCode
		}
	}

	if rmcResponseBody == nil {
		pSearchResult, errCode := database.SearchObjectsByDataStoreSearchParam(manager.Database, manager, connection.PID(), param, totalCountEnabled)
		if errCode != nil {
			return nil, errCode
		}

		rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

		pSearchResult.WriteTo(rmcResponseStream)

		rmcResponseBody = rmcResponseStream.Bytes()

		if param.UseCache {
			errCode = database.InsertCachedSearchObjectResponse(manager.Database, manager, connection.PID(), cacheKey, rmcResponseBody)
			if errCode != nil {
				return nil, errCode
			}
		}
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodSearchObject
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterSearchObject != nil {
		go commonProtocol.OnAfterSearchObject(packet, param)
	}

	return rmcResponse, nil
}
