package common_globals

import (
	pb "github.com/PretendoNetwork/grpc/go/account/v2"
	"github.com/PretendoNetwork/plogger-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var Logger = plogger.NewLogger()
var GRPCAccountClientConnection *grpc.ClientConn
var GRPCAccountClient pb.AccountServiceClient
var GRPCAccountCommonMetadata metadata.MD
