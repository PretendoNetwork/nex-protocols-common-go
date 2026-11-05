package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) searchObjectLight(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreSearchParam) (*nex.RMCMessage, *nex.Error) {
	// * We don't know how these methods differ, so fuck it. We ball, reuse the non-light version
	// TODO - Check on the Switch servers to see how these differ
	rmcResponse, errCode := commonProtocol.searchObject(err, packet, callID, param)
	if errCode != nil {
		return nil, errCode
	}

	rmcResponse.MethodID = datastore.MethodSearchObjectLight

	return rmcResponse, nil
}
