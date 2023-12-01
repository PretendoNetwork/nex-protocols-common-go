package datastore

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func rateObjects(err error, packet nex.PacketInterface, callID uint32, targets []*datastore_types.DataStoreRatingTarget, params []*datastore_types.DataStoreRateObjectParam, transactional bool, fetchRatings bool) (*nex.RMCMessage, uint32) {
	if commonDataStoreProtocol.GetObjectInfoByDataIDWithPassword == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataIDWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.RateObjectWithPassword == nil {
		common_globals.Logger.Warning("RateObjectWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	client := packet.Sender().(*nex.PRUDPClient)

	pRatings := make([]*datastore_types.DataStoreRatingInfo, 0)
	pResults := make([]*nex.Result, 0)

	// * Real DataStore does not actually check this.
	// * I just didn't feel like working out the
	// * logic for differing sized lists. So force
	// * them to always be the same
	if len(targets) != len(params) {
		return nil, nex.Errors.DataStore.InvalidArgument
	}

	for i := 0; i < len(targets); i++ {
		target := targets[i]
		param := params[i]

		objectInfo, errCode := commonDataStoreProtocol.GetObjectInfoByDataIDWithPassword(target.DataID, param.AccessPassword)
		if errCode != 0 {
			return nil, errCode
		}

		errCode = commonDataStoreProtocol.VerifyObjectPermission(objectInfo.OwnerID, client.PID(), objectInfo.Permission)
		if errCode != 0 {
			return nil, errCode
		}

		rating, errCode := commonDataStoreProtocol.RateObjectWithPassword(target.DataID, target.Slot, param.RatingValue, param.AccessPassword)
		if errCode != 0 {
			return nil, errCode
		}

		if fetchRatings {
			pRatings = append(pRatings, rating)
		}
	}

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	nex.StreamWriteListStructure(rmcResponseStream, pRatings)
	rmcResponseStream.WriteListResult(pResults) // * pResults is ALWAYS empty in SMM?

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodRateObjects
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
