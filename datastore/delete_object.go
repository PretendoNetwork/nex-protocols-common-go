package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func deleteObject(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStoreDeleteParam) uint32 {
	if commonDataStoreProtocol.getObjectInfoByDataIDHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.deleteObjectByDataIDWithPasswordHandler == nil {
		common_globals.Logger.Warning("DeleteObjectByDataIDWithPassword not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	client := packet.Sender().(*nex.PRUDPClient)

	metaInfo, errCode := commonDataStoreProtocol.getObjectInfoByDataIDHandler(param.DataID)
	if errCode != 0 {
		return errCode
	}

	errCode = commonDataStoreProtocol.VerifyObjectPermission(metaInfo.OwnerID, client.PID(), metaInfo.DelPermission)
	if errCode != 0 {
		return errCode
	}

	errCode = commonDataStoreProtocol.deleteObjectByDataIDWithPasswordHandler(param.DataID, param.UpdatePassword)
	if errCode != 0 {
		return errCode
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodDeleteObject
	rmcResponse.CallID = callID

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if commonDataStoreProtocol.server.PRUDPVersion == 0 {
		responsePacket, _ = nex.NewPRUDPPacketV0(client, nil)
	} else {
		responsePacket, _ = nex.NewPRUDPPacketV1(client, nil)
	}

	responsePacket.SetType(nex.DataPacket)
	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)
	responsePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	responsePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	responsePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	responsePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	responsePacket.SetPayload(rmcResponseBytes)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
