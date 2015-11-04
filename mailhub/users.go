package mailhub

import (
	"github.com/gschat/gschat"
	"github.com/gsrpc/gorpc"
)

type _UserResolver struct {
	resolver gschat.UserResolver
}

func (mailhub *_MailHub) newUserResolver(resolver gschat.UserResolver) *_UserResolver {
	return &_UserResolver{
		resolver: resolver,
	}
}

func (resolver *_UserResolver) GroupChanged(callSite *gorpc.CallSite, groupID string) (err error) {
	return nil
}

func (resolver *_UserResolver) GroupRemoved(callSite *gorpc.CallSite, groupID string) (err error) {
	return nil
}

func (resolver *_UserResolver) BlockRuleChanged(callSite *gorpc.CallSite, userID string) (err error) {
	return nil
}
