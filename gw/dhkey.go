package gw

import (
	"math/big"

	"github.com/gschat/gschat"
	"github.com/gsdocker/gsconfig"
	"github.com/gsdocker/gserrors"
	"github.com/gsrpc/gorpc"
	"github.com/gsrpc/gorpc/handler"
)

type _DHKeyResolver struct {
	improxy *IMProxy
}

// NewDHKeyResolver create new dhkey resolver
func (improxy *IMProxy) NewDHKeyResolver() handler.DHKeyResolver {
	return &_DHKeyResolver{
		improxy: improxy,
	}
}

func (resolver *_DHKeyResolver) Resolve(device *gorpc.Device) (retval *handler.DHKey, err error) {

	var dhkeyStr *gschat.DHKey

	if service, ok := resolver.improxy.service(gschat.NameOfDHKeyResolver, device.String()); ok {
		dhkeyStr, err = gschat.BindDHKeyResolver(uint16(gschat.ServiceDHKeyResolver), service).DHKeyResolve(nil, device)

		if err != nil {
			return nil, err
		}

	} else {
		gStr := gsconfig.String("gsproxy.dhkey.G", "6849211231874234332173554215962568648211715948614349192108760170867674332076420634857278025209099493881977517436387566623834457627945222750416199306671083")

		pStr := gsconfig.String("gsproxy.dhkey.P", "13196520348498300509170571968898643110806720751219744788129636326922565480984492185368038375211941297871289403061486510064429072584259746910423138674192557")

		dhkeyStr = &gschat.DHKey{G: gStr, P: pStr}

		// G, _ := new(big.Int).SetString(gStr, 0)
		//
		// P, _ := new(big.Int).SetString(pStr, 0)
		//
		// return handler.NewDHKey(G, P), nil
	}

	G, ok := new(big.Int).SetString(dhkeyStr.G, 0)

	if !ok {
		return nil, gserrors.Newf(nil, "invalid G :%s", dhkeyStr.G)
	}

	P, ok := new(big.Int).SetString(dhkeyStr.P, 0)

	if !ok {
		return nil, gserrors.Newf(nil, "invalid P :%s", dhkeyStr.P)
	}

	return handler.NewDHKey(G, P), nil

}
