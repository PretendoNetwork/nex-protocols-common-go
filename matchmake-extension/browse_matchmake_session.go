package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) browseMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, searchCriteria match_making_types.MatchmakeSessionSearchCriteria, resultRange types.ResultRange) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.RLock()

	searchCriterias := []match_making_types.MatchmakeSessionSearchCriteria{searchCriteria}

	lstSearchCriteria := types.NewList[match_making_types.MatchmakeSessionSearchCriteria]()
	lstSearchCriteria = searchCriterias

	if commonProtocol.CleanupMatchmakeSessionSearchCriterias != nil {
		commonProtocol.CleanupMatchmakeSessionSearchCriterias(lstSearchCriteria)
	}

	sessions, nexError := database.FindMatchmakeSessionBySearchCriteria(commonProtocol.manager, connection, searchCriterias, resultRange, nil)
	if nexError != nil {
		commonProtocol.manager.Mutex.RUnlock()
		return nil, nexError
	}

	lstGathering := types.NewList[match_making_types.GatheringHolder]()

	for _, session := range sessions {
		// * Scrap session key and user password
		session.SessionKey = make([]byte, 0)
		session.UserPassword = ""

		matchmakeSessionDataHolder := match_making_types.NewGatheringHolder()
		matchmakeSessionDataHolder.Object = session.Copy().(match_making_types.GatheringInterface)

		lstGathering = append(lstGathering, matchmakeSessionDataHolder)
	}

	commonProtocol.manager.Mutex.RUnlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	lstGathering.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodBrowseMatchmakeSession
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterBrowseMatchmakeSession != nil {
		go commonProtocol.OnAfterBrowseMatchmakeSession(packet, searchCriteria, resultRange)
	}

	return rmcResponse, nil
}
