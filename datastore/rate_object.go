package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func rateObject(err error, packet nex.PacketInterface, callID uint32, target *datastore_types.DataStoreRatingTarget, param *datastore_types.DataStoreRateObjectParam, fetchRatings bool) (*nex.RMCMessage, uint32) {
	if commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataIDWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.rateObjectWithPasswordHandler == nil {
		common_globals.Logger.Warning("RateObjectWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	client := packet.Sender()

	objectInfo, errCode := commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler(target.DataID, param.AccessPassword)
	if errCode != 0 {
		return nil, errCode
	}

	errCode = commonDataStoreProtocol.VerifyObjectPermission(objectInfo.OwnerID, client.PID(), objectInfo.Permission)
	if errCode != 0 {
		return nil, errCode
	}

	pRating, errCode := commonDataStoreProtocol.rateObjectWithPasswordHandler(target.DataID, target.Slot, param.RatingValue, param.AccessPassword)
	if errCode != 0 {
		return nil, errCode
	}

	// * This is kinda backwards. Server returns
	// * the rating by default, so we check if
	// * the client DOESN'T want it and then just
	// * zero it out
	if !fetchRatings {
		pRating = datastore_types.NewDataStoreRatingInfo()
	}

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	rmcResponseStream.WriteStructure(pRating)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodRateObject
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
