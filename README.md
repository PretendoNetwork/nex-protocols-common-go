# NEX Protocols Common Go
## NEX protocols used by many games with premade handlers and a high level API

[![GoDoc](https://godoc.org/github.com/PretendoNetwork/nex-protocols-common-go?status.svg)](https://godoc.org/github.com/PretendoNetwork/nex-protocols-common-go)

### Other NEX libraries
[nex-go](https://github.com/PretendoNetwork/nex-go) - Barebones NEX/PRUDP server implementation

[nex-protocols-go](https://github.com/PretendoNetwork/nex-protocols-go) - NEX protocol definitions

### Install

`go get github.com/PretendoNetwork/nex-protocols-common-go`

### Usage

`nex-protocols-common-go` provides a higher level API than the [NEX Protocols Go module](https://github.com/PretendoNetwork/nex-protocols-go). This module handles many of the more common protcols and methods used shared by many servers. Instead of working directly with the NEX server, this module exposes an API for defining helper functions to provide the module with the data it needs to run

### Example, friends (Wii U) authentication server
### For a complete example, see the complete [Friends Authentication Server](https://github.com/PretendoNetwork/friends-authentication), and other game servers

```go
package main

import (
	"fmt"
	"os"

	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-protocols-common-go/authentication"
)

var nexServer *nex.Server

func main() {
	nexServer = nex.NewServer()
	nexServer.SetPRUDPVersion(0)
	nexServer.SetKerberosKeySize(16)
	nexServer.SetKerberosPassword(os.Getenv("KERBEROS_PASSWORD"))
	nexServer.SetAccessKey("ridfebb9")

	nexServer.On("Data", func(packet *nex.PacketV0) {
		request := packet.RMCRequest()

		fmt.Println("==Friends - Auth==")
		fmt.Printf("Protocol ID: %#v\n", request.ProtocolID())
		fmt.Printf("Method ID: %#v\n", request.MethodID())
		fmt.Println("==================")
	})

	authenticationProtocol := authentication.NewCommonAuthenticationProtocol(nexServer)

	secureStationURL := nex.NewStationURL("")
	secureStationURL.SetScheme("prudps")
	secureStationURL.SetAddress(os.Getenv("SECURE_SERVER_LOCATION"))
	secureStationURL.SetPort(os.Getenv("SECURE_SERVER_PORT"))
	secureStationURL.SetCID("1")
	secureStationURL.SetPID("2")
	secureStationURL.SetSID("1")
	secureStationURL.SetStream("10")
	secureStationURL.SetType("2")

	authenticationProtocol.SetSecureStationURL(secureStationURL)
	authenticationProtocol.SetBuildName("Pretendo Friends Auth")
	authenticationProtocol.SetPasswordFromPIDFunction(passwordFromPID)

	nexServer.Listen(":60000")
}
```