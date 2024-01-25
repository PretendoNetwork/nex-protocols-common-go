package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func rateObjects(err error, packet nex.PacketInterface, callID uint32, targets *types.List[*datastore_types.DataStoreRatingTarget], params *types.List[*datastore_types.DataStoreRateObjectParam], transactional *types.PrimitiveBool, fetchRatings *types.PrimitiveBool) (*nex.RMCMessage, uint32) {
	if commonProtocol.GetObjectInfoByDataIDWithPassword == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataIDWithPassword not defined")
		return nil, nex.ResultCodes.Core.NotImplemented
	}

	if commonProtocol.RateObjectWithPassword == nil {
		common_globals.Logger.Warning("RateObjectWithPassword not defined")
		return nil, nex.ResultCodes.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.DataStore.Unknown
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	pRatings := types.NewList[*datastore_types.DataStoreRatingInfo]()
	pResults := types.NewList[*types.QResult]()

	pRatings.Type = datastore_types.NewDataStoreRatingInfo()
	pResults.Type = types.NewQResult(0)

	// * Real DataStore does not actually check this.
	// * I just didn't feel like working out the
	// * logic for differing sized lists. So force
	// * them to always be the same
	if targets.Length() != params.Length() {
		return nil, nex.ResultCodes.DataStore.InvalidArgument
	}

	var errorCode uint32

	targets.Each(func(i int, target *datastore_types.DataStoreRatingTarget) bool {
		param, err := params.Get(i)
		if err != nil {
			errorCode = nex.ResultCodes.DataStore.InvalidArgument
			return true
		}

		objectInfo, errCode := commonProtocol.GetObjectInfoByDataIDWithPassword(target.DataID, param.AccessPassword)
		if errCode != 0 {
			errorCode = errCode
			return true
		}

		errCode = commonProtocol.VerifyObjectPermission(objectInfo.OwnerID, connection.PID(), objectInfo.Permission)
		if errCode != 0 {
			errorCode = errCode
			return true
		}

		rating, errCode := commonProtocol.RateObjectWithPassword(target.DataID, target.Slot, param.RatingValue, param.AccessPassword)
		if errCode != 0 {
			errorCode = errCode
			return true
		}

		if fetchRatings.Value {
			pRatings.Append(rating)
		}

		return false
	})

	if errorCode != 0 {
		return nil, errorCode
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	pRatings.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream) // * pResults is ALWAYS empty in SMM?

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodRateObjects
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
