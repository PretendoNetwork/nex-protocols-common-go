package utility

import (
	nex "github.com/PretendoNetwork/nex-go"
	utility "github.com/PretendoNetwork/nex-protocols-go/utility"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func acquireNexUniqueID(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	if commonUtilityProtocol.GenerateNEXUniqueID == nil {
		common_globals.Logger.Warning("Utility::AcquireNexUniqueID missing GenerateNEXUniqueID!")
		return nil, nex.Errors.Core.NotImplemented
	}

	pNexUniqueID := commonUtilityProtocol.GenerateNEXUniqueID()

	server := commonUtilityProtocol.server

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteUInt64LE(pNexUniqueID)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodAcquireNexUniqueID
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
