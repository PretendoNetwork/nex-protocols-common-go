package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func rateObject(err error, packet nex.PacketInterface, callID uint32, target *datastore_types.DataStoreRatingTarget, param *datastore_types.DataStoreRateObjectParam, fetchRatings *types.PrimitiveBool) (*nex.RMCMessage, uint32) {
	if commonProtocol.GetObjectInfoByDataIDWithPassword == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataIDWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonProtocol.RateObjectWithPassword == nil {
		common_globals.Logger.Warning("RateObjectWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	objectInfo, errCode := commonProtocol.GetObjectInfoByDataIDWithPassword(target.DataID, param.AccessPassword)
	if errCode != 0 {
		return nil, errCode
	}

	errCode = commonProtocol.VerifyObjectPermission(objectInfo.OwnerID, connection.PID(), objectInfo.Permission)
	if errCode != 0 {
		return nil, errCode
	}

	pRating, errCode := commonProtocol.RateObjectWithPassword(target.DataID, target.Slot, param.RatingValue, param.AccessPassword)
	if errCode != 0 {
		return nil, errCode
	}

	// * This is kinda backwards. Server returns
	// * the rating by default, so we check if
	// * the client DOESN'T want it and then just
	// * zero it out
	if !fetchRatings.Value {
		pRating = datastore_types.NewDataStoreRatingInfo()
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	pRating.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodRateObject
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
