package datastore

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func rateObjects(err error, client *nex.Client, callID uint32, targets []*datastore_types.DataStoreRatingTarget, params []*datastore_types.DataStoreRateObjectParam, transactional bool, fetchRatings bool) uint32 {
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

	pRatings := make([]*datastore_types.DataStoreRatingInfo, 0)
	pResults := make([]*nex.Result, 0)

	// * Real DataStore does not actually check this.
	// * I just didn't feel like working out the
	// * logic for differing sized lists. So force
	// * them to always be the same
	if len(targets) != len(params) {
		return nex.Errors.DataStore.InvalidArgument
	}

	for i := 0; i < len(targets); i++ {
		target := targets[i]
		param := params[i]

		objectInfo, errCode := commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler(target.DataID, param.AccessPassword)
		if errCode != 0 {
			return errCode
		}

		errCode = commonDataStoreProtocol.verifyObjectPermissionHandler(objectInfo.OwnerID, client.PID(), objectInfo.Permission)
		if errCode != 0 {
			return errCode
		}

		rating, errCode := commonDataStoreProtocol.rateObjectWithPasswordHandler(target.DataID, target.Slot, param.RatingValue, param.AccessPassword)
		if errCode != 0 {
			return errCode
		}

		if fetchRatings {
			pRatings = append(pRatings, rating)
		}
	}

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	rmcResponseStream.WriteListStructure(pRatings)
	rmcResponseStream.WriteListResult(pResults) // * pResults is ALWAYS empty in SMM?

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodRateObjects, rmcResponseBody)

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
