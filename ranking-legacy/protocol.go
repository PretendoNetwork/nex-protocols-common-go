package ranking_legacy

import (
	"github.com/PretendoNetwork/nex-go/v2"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	rankinglegacy "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy"
)

type CommonProtocol struct {
	endpoint nex.EndpointInterface
	protocol rankinglegacy.Interface
	manager  *commonglobals.RankingManager
}

// SetManager defines the utility manager to be used by the common protocol
func (commonProtocol *CommonProtocol) SetManager(manager *commonglobals.RankingManager) {
	var err error

	commonProtocol.manager = manager

	_, err = manager.Database.Exec(`CREATE SCHEMA IF NOT EXISTS ranking_legacy`)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS ranking_legacy.scores (
    	deleted boolean NOT NULL DEFAULT FALSE,
    	create_time timestamptz,
    	update_time timestamptz,

    	owner_pid numeric(20) /* uint8 */,
    	unique_id int8 /* uint4 */,
    	category int8 /* uint4 */,

    	scores int8 ARRAY[2] /* uint4[2] */,
    	unk1 int2 /* uint1 */,
    	unk2 int8 /* uint4 */,
    	PRIMARY KEY (owner_pid, unique_id, category)
	)`)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS ranking_legacy.common_data (
    	deleted boolean NOT NULL DEFAULT FALSE,
		create_time timestamptz,
    	update_time timestamptz,

    	owner_pid numeric(20) /* uint8 */,
    	unique_id int8 /* uint4 */,

    	data bytea,
        PRIMARY KEY (owner_pid, unique_id)
	)`)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol rankinglegacy.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerUploadCommonData(commonProtocol.uploadCommonData)
	protocol.SetHandlerGetCommonData(commonProtocol.getCommonData)
	protocol.SetHandlerDeleteCommonData(commonProtocol.deleteCommonData)
	protocol.SetHandlerUploadScore(commonProtocol.uploadScore)
	protocol.SetHandlerUploadScoreWithLimit(commonProtocol.uploadScoreWithLimit)
	protocol.SetHandlerUploadScores(commonProtocol.uploadScores)
	protocol.SetHandlerUploadScoresWithLimit(commonProtocol.uploadScoresWithLimit)
	protocol.SetHandlerDeleteScore(commonProtocol.deleteScore)
	protocol.SetHandlerDeleteAllScore(commonProtocol.deleteAllScore)
	protocol.SetHandlerGetScore(commonProtocol.getScore)
	protocol.SetHandlerGetSelfScore(commonProtocol.getSelfScore)
	protocol.SetHandlerGetTopScore(commonProtocol.getTopScore)
	protocol.SetHandlerUnk0xD(commonProtocol.getSelfRankingOrder)

	return commonProtocol
}
