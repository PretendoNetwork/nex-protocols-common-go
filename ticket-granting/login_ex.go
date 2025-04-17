package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"
)

func (commonProtocol *CommonProtocol) loginEx(err error, packet nex.PacketInterface, callID uint32, strUserName types.String, oExtraData types.DataHolder) (*nex.RMCMessage, *nex.Error) {
	if commonProtocol.ValidateLoginData == nil {
		common_globals.Logger.Error("TicketGranting::LoginEx missing ValidateLoginData!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "TicketGranting::LoginEx missing ValidateLoginData!")
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	sourceAccount, errorCode := endpoint.AccountDetailsByUsername(string(strUserName))

	if errorCode == nil {
		// * The connection doesn't have a PID set here, so we use the source PID
		errorCode = commonProtocol.ValidateLoginData(sourceAccount.PID, oExtraData)
	}

	var targetAccount *nex.Account
	if errorCode == nil {
		targetAccount, errorCode = endpoint.AccountDetailsByUsername(commonProtocol.SecureServerAccount.Username)
	}

	var encryptedTicket []byte
	if errorCode == nil {
		encryptedTicket, errorCode = generateTicket(sourceAccount, targetAccount, commonProtocol.SessionKeyLength, endpoint)
	}

	var retval types.QResult
	pidPrincipal := types.NewPID(0)
	pbufResponse := types.NewBuffer([]byte{})
	pConnectionData := types.NewRVConnectionData()
	strReturnMsg := types.NewString("")

	// * If any errors are triggered, return them in %retval%
	if errorCode != nil {
		common_globals.Logger.Error(errorCode.Message)
		retval = types.NewQResultError(errorCode.ResultCode)
	} else {
		retval = types.NewQResultSuccess(nex.ResultCodes.Core.Unknown)
		pidPrincipal = sourceAccount.PID
		pbufResponse = types.NewBuffer(encryptedTicket)
		strReturnMsg = commonProtocol.BuildName.Copy().(types.String)

		specialProtocols := types.NewList[types.UInt8]()
		specialProtocols = commonProtocol.SpecialProtocols

		pConnectionData.StationURL = commonProtocol.SecureStationURL
		pConnectionData.SpecialProtocols = specialProtocols
		pConnectionData.StationURLSpecialProtocols = commonProtocol.StationURLSpecialProtocols
		pConnectionData.Time = types.NewDateTime(0).Now()

		if endpoint.LibraryVersions().Main.GreaterOrEqual("v3.5.0") {
			pConnectionData.StructureVersion = 1
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)
	pidPrincipal.WriteTo(rmcResponseStream)
	pbufResponse.WriteTo(rmcResponseStream)
	pConnectionData.WriteTo(rmcResponseStream)
	strReturnMsg.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodLoginEx
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterLoginEx != nil {
		go commonProtocol.OnAfterLoginEx(packet, strUserName, oExtraData)
	}

	return rmcResponse, nil
}
