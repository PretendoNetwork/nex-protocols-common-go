package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"
)

func login(err error, packet nex.PacketInterface, callID uint32, strUserName *types.String) (*nex.RMCMessage, uint32) {
	if !commonProtocol.allowInsecureLoginMethod {
		return nil, nex.ResultCodes.Authentication.ValidationFailed
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	sourceAccount, errorCode := commonProtocol.AccountDetailsByUsername(strUserName.Value)
	if errorCode != 0 && errorCode != nex.ResultCodes.RendezVous.InvalidUsername {
		// * Some other error happened
		return nil, errorCode
	}

	targetAccount, errorCode := commonProtocol.AccountDetailsByUsername(commonProtocol.SecureServerAccount.Username)
	if errorCode != 0 && errorCode != nex.ResultCodes.RendezVous.InvalidUsername {
		// * Some other error happened
		return nil, errorCode
	}

	encryptedTicket, errorCode := generateTicket(sourceAccount, targetAccount, commonProtocol.SessionKeyLength, server)

	if errorCode != 0 && errorCode != nex.ResultCodes.RendezVous.InvalidUsername {
		// * Some other error happened
		return nil, errorCode
	}

	var retval *types.QResult
	pidPrincipal := types.NewPID(0)
	pbufResponse := types.NewBuffer([]byte{})
	pConnectionData := types.NewRVConnectionData()
	strReturnMsg := types.NewString("")

	// * From the wiki:
	// *
	// * "If the username does not exist, the %retval% field is set to
	// * RendezVous::InvalidUsername and the other fields are left blank."
	if errorCode == nex.ResultCodes.RendezVous.InvalidUsername {
		retval = types.NewQResultError(errorCode)
	} else {
		retval = types.NewQResultSuccess(nex.ResultCodes.Core.Unknown)
		pidPrincipal = sourceAccount.PID
		pbufResponse = types.NewBuffer(encryptedTicket)
		strReturnMsg = commonProtocol.BuildName.Copy().(*types.String)

		specialProtocols := types.NewList[*types.PrimitiveU8]()

		specialProtocols.Type = types.NewPrimitiveU8(0)
		specialProtocols.SetFromData(commonProtocol.SpecialProtocols)

		pConnectionData.StationURL = commonProtocol.SecureStationURL
		pConnectionData.SpecialProtocols = specialProtocols
		pConnectionData.StationURLSpecialProtocols = commonProtocol.StationURLSpecialProtocols
		pConnectionData.Time = types.NewDateTime(0).Now()
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	retval.WriteTo(rmcResponseStream)
	pidPrincipal.WriteTo(rmcResponseStream)
	pbufResponse.WriteTo(rmcResponseStream)
	pConnectionData.WriteTo(rmcResponseStream)
	strReturnMsg.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodLogin
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
