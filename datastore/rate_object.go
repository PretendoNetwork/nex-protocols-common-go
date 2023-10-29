package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func rateObject(err error, client *nex.Client, callID uint32, target *datastore_types.DataStoreRatingTarget, param *datastore_types.DataStoreRateObjectParam, fetchRatings bool) uint32 {
	if commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataIDWithPassword not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.verifyObjectPermissionHandler == nil {
		common_globals.Logger.Warning("VerifyObjectPermission not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.rateObjectWithPasswordHandler == nil {
		common_globals.Logger.Warning("RateObjectWithPassword not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	objectInfo, errCode := commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler(target.DataID, param.AccessPassword)
	if errCode != 0 {
		return errCode
	}

	errCode = commonDataStoreProtocol.verifyObjectPermissionHandler(objectInfo.OwnerID, client.PID(), objectInfo.Permission)
	if errCode != 0 {
		return errCode
	}

	pRating, errCode := commonDataStoreProtocol.rateObjectWithPasswordHandler(target.DataID, target.Slot, param.RatingValue, param.AccessPassword)
	if errCode != 0 {
		return errCode
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

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodRateObject, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if commonDataStoreProtocol.server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
