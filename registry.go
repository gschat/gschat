package gschat

import (
	"github.com/gsrpc/gorpc"
)

var regsitry = map[string]uint16{
	NameOfMailHub:              uint16(ServiceMailHub),
	NameOfPush:                 uint16(ServicePush),
	NameOfAuth:                 uint16(ServiceAuth),
	NameOfClient:               uint16(ServiceClient),
	NameOfUserBinder:           uint16(ServiceUserBinder),
	NameOfPushServiceProvider:  uint16(ServicePushServiceProvider),
	NameOfDHKeyResolver:        uint16(ServiceDHKeyResolver),
	NameOfUserResolverListener: uint16(ServiceUserResolverListener),
	NameOfUserResolver:         uint16(ServiceUserResolver),
	NameOfGateway:              uint16(ServiceGateway),
}

func init() {
	gorpc.RegistryUpdate(regsitry)
}
