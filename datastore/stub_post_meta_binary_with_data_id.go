package datastore

import (
	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) stubPostMetaBinaryWithDataID(err error, packet nex.PacketInterface, callID uint32, dataID types.UInt64, param datastore_types.DataStorePreparePostParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	// TODO - Implement this once we see a client use it
	// * This method seems to allow for uploading an object using a predetermined DataID.
	// * It is unclear if this is intended for retail clients or only admin use, so leaving
	// * as stubbed until we see a client use it to gather more information.
	// *
	// * Some notes:
	// *
	// * In Xenoblade we have the following 2 functions:
	// *
	// * PostObject__Q3_2nn3nex74DataStoreClientTemplate__tm__43_Q3_2nn3nex30DataStoreLogicServerHttpClientFPQ3_2nn3nex19ProtocolCallContextULRCQ3_2nn3nex25DataStorePreparePostParam_b
	// * PostObject__Q3_2nn3nex70DataStoreClientTemplate__tm__39_Q3_2nn3nex26DataStoreLogicServerClientFPQ3_2nn3nex19ProtocolCallContextULRCQ3_2nn3nex25DataStorePreparePostParam_b
	// *
	// * These unmangle to:
	// *
	// * bool nn::nex::DataStoreClientTemplate<class Z1 = nn::nex::DataStoreLogicServerHttpClient>::PostObject(nn::nex::ProtocolCallContext *, unsigned long long, nn::nex::DataStorePreparePostParam const &)
	// * bool nn::nex::DataStoreClientTemplate<class Z1 = nn::nex::DataStoreLogicServerClient>::PostObject(nn::nex::ProtocolCallContext *, unsigned long long, nn::nex::DataStorePreparePostParam const &)
	// *
	// * Based on the name of this method, and the presence of the `unsigned long long` parameter,
	// * which is the same size as a DataID, I believe this method is used to upload objects
	// * using a predetermined DataID. That being said, I do not believe that the predetermined DataID
	// * can be ANYTHING. I believe that it's likely that this was used to upload objects into
	// * the 900,000-999,999 range. This is based on 3 things:
	// *
	// * 1. We already know that this range is special, and has unique handling. See methods
	// *    like `DeleteObject` for an example
	// * 2. We know for a FACT that ALL objects uploaded through normal means start at DataID
	// *    1,000,000 and count up sequentially. If objects can be created with a predetermined
	// *    DataID, it would only make sense for that to be limited to the 900,000-999,999 range
	// *    as to not interfere with normal objects
	// * 3. We can observe in Super Mario Maker how Nintendo was able to push new objects into
	// *    the 900,000-999,999 range while the 1,000,000+ range is counting still, unaffected
	// *
	// * We can also observe in Super Mario Maker how objects in this range have both their
	// * permissions set to "PERMISSION_PUBLIC", yet we cannot update/delete data in this range
	// * despite the permissions, and the owner is always PID 2 (the server). This makes me believe
	// * that this was only intended for admin/system use. However, once we implement this I think
	// * it would be wise to "convert" data uploaded here into "system data". This means changing
	// * the owner PID of the object to that of the server, making it never expire, ignoring
	// * the persistence slot (because it's in the "system" DataID range), etc.

	return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "PostMetaBinaryWithDataID is not implemented")
}
