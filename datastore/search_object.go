package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func searchObject(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStoreSearchParam) uint32 {
	if commonDataStoreProtocol.getObjectInfosByDataStoreSearchParamHandler == nil {
		common_globals.Logger.Warning("GetObjectInfosByDataStoreSearchParam not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	client := packet.Sender()

	// * This is likely game-specific. Also developer note:
	// * Please keep in mind that no results is allowed. errCode
	// * should NEVER be DataStore::NotFound!
	// *
	// * DataStoreSearchParam contains a ResultRange to limit the
	// * returned results. TotalCount is the total matching objects
	// * in the database, whereas objects is the limited results
	objects, totalCount, errCode := commonDataStoreProtocol.getObjectInfosByDataStoreSearchParamHandler(param)
	if errCode != 0 {
		return errCode
	}

	pSearchResult := datastore_types.NewDataStoreSearchResult()

	pSearchResult.Result = make([]*datastore_types.DataStoreMetaInfo, 0, len(objects))

	for _, object := range objects {
		errCode = commonDataStoreProtocol.VerifyObjectPermission(object.OwnerID, client.PID(), object.Permission)
		if errCode != 0 {
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
	if param.StructureVersion() >= 3 || commonDataStoreProtocol.server.DataStoreProtocolVersion().GreaterOrEqual("4.0.0") {
		if !param.TotalCountEnabled {
			totalCount = 0
			totalCountType = 3
		}
	}

	pSearchResult.TotalCount = totalCount
	pSearchResult.TotalCountType = totalCountType

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	rmcResponseStream.WriteStructure(pSearchResult)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodSearchObject, rmcResponseBody)

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
