package match_making_ext

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/v2/match-making-ext"
)

type CommonProtocol struct {
	endpoint                nex.EndpointInterface
	protocol                match_making_ext.Interface
	db                      *sql.DB
	OnAfterEndParticipation func(acket nex.PacketInterface, idGathering *types.PrimitiveU32, strMessage *types.String)
}

// SetDatabase defines the SQL database to be used by the common protocol
func (commonProtocol *CommonProtocol) SetDatabase(db *sql.DB) {
	commonProtocol.db = db
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol match_making_ext.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerEndParticipation(commonProtocol.endParticipation)

	return commonProtocol
}
