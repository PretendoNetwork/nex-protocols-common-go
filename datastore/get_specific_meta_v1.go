package datastore

import (
	"math"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) getSpecificMetaV1(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreGetSpecificMetaParamV1) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pMetaInfos := types.NewList[datastore_types.DataStoreSpecificMetaInfoV1]()
	dataIDs := param.DataIDs.Copy().(types.List[types.UInt32])
	slices.Sort(dataIDs)

	for _, dataID := range dataIDs {
		metaInfo, _, errCode := database.GetAccessObjectInfoByDataID(manager.Database, types.UInt64(dataID))
		if errCode != nil {
			// * Unknown objects of any kind are just not sent back
			if errCode.Equals(nex.ResultCodes.DataStore.NotFound) {
				continue
			}

			return nil, errCode
		}

		version, errCode := database.GetObjectLatestVersionNumber(manager.Database, metaInfo.DataID)
		if errCode != nil {
			return nil, errCode
		}

		if version > math.MaxUint16 {
			// * OverCapacity is definitely not the correct error here, but it sounds nice
			return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
		}

		specificMetaInfo := datastore_types.NewDataStoreSpecificMetaInfoV1()
		specificMetaInfo.DataID = types.UInt32(metaInfo.DataID)
		specificMetaInfo.OwnerID = metaInfo.OwnerID
		specificMetaInfo.Size = metaInfo.Size
		specificMetaInfo.DataType = metaInfo.DataType
		specificMetaInfo.Version = types.UInt16(version)

		pMetaInfos = append(pMetaInfos, specificMetaInfo)
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pMetaInfos.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetSpecificMetaV1
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
