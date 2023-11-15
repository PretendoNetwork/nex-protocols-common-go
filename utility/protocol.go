package utility

import (
	"math/rand"
	"time"

	nex "github.com/PretendoNetwork/nex-go"
	utility "github.com/PretendoNetwork/nex-protocols-go/utility"
)

var commonUtilityProtocol *CommonUtilityProtocol

type CommonUtilityProtocol struct {
	*utility.Protocol
	server           *nex.PRUDPServer
	randSource       rand.Source
	randGenerator    *rand.Rand
	randomU64Handler func() uint64
}

// RandomU64 sets the RandomU64 handler function
func (c *CommonUtilityProtocol) RandomU64(handler func() uint64) {
	c.randomU64Handler = handler
}

// NewCommonUtilityProtocol returns a new CommonUtilityProtocol
func NewCommonUtilityProtocol(server *nex.PRUDPServer) *CommonUtilityProtocol {
	utilityProtocol := utility.NewProtocol(server)
	commonUtilityProtocol = &CommonUtilityProtocol{Protocol: utilityProtocol, server: server}

	// * These are used as defaults for if randomU64Handler is not set
	commonUtilityProtocol.randSource = rand.NewSource(time.Now().Unix())
	commonUtilityProtocol.randGenerator = rand.New(commonUtilityProtocol.randSource)

	// TODO - Organize these by method ID
	commonUtilityProtocol.AcquireNexUniqueID = acquireNexUniqueID

	return commonUtilityProtocol
}
