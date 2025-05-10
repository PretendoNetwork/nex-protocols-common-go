package datastore

import (
	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) stubPostMetaBinariesWithDataID(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64], params types.List[datastore_types.DataStorePreparePostParam], transactional types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	// TODO - Implement this once we see a client use it
	// * See PostMetaBinaryWithDataID for details

	return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "PostMetaBinariesWithDataID is not implemented")
}
