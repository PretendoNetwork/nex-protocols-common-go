package datastore

import (
	"math"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) getSpecificMeta(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreGetSpecificMetaParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pMetaInfos := types.NewList[datastore_types.DataStoreSpecificMetaInfo]()
	dataIDs := param.DataIDs.Copy().(types.List[types.UInt64])
	slices.Sort(dataIDs)

	dataIDs = slices.Compact(dataIDs)

	for _, dataID := range dataIDs {
		metaInfo, version, errCode := database.GetSpecificObjectInfoByDataID(manager.Database, dataID)
		if errCode != nil {
			// * Unknown objects of any kind are just not sent back
			if errCode.Equals(nex.ResultCodes.DataStore.NotFound) {
				continue
			}

			return nil, errCode
		}

		size := metaInfo.Size

		// * Objects that don't use the file server only exist in the database and use
		// * the MetaBinary size instead. Or 0xFFFFFFFF if the size is 0, for some reason.
		// * Don't ask, this is what the real servers do
		if metaInfo.Flag.HasFlag(datastore_constants.DataFlagNotUseFileServer) {
			if len(metaInfo.MetaBinary) == 0 {
				size = math.MaxUint32
			} else {
				size = types.UInt32(len(metaInfo.MetaBinary))
			}
		}

		specificMetaInfo := datastore_types.NewDataStoreSpecificMetaInfo()
		specificMetaInfo.DataID = metaInfo.DataID
		specificMetaInfo.OwnerID = metaInfo.OwnerID
		specificMetaInfo.Size = size
		specificMetaInfo.DataType = metaInfo.DataType
		specificMetaInfo.Version = types.UInt32(version)

		pMetaInfos = append(pMetaInfos, specificMetaInfo)
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pMetaInfos.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetSpecificMeta
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
