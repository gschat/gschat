package com.gridy.soa;

using com.gsrpc.Device;
using gslang.Exception;

@Package(Lang:"golang",Name:"com.gschat",Redirect:"github.com/gschat/gschat/gridy")

@Exception
table NotFound {}

table DeviceToken {
    string P;
    string G;
}

enum Service{
    Business,IM
}

contract GridyBusiness {
    // query device access token
    DeviceToken Token(Device device) throws(NotFound);
    // query user group list
    uint32[] Group(uint32 groupID) throws(NotFound);
    // query user block list
    uint32[] UserBlockList(uint32 userid) throws(NotFound);
}

contract GridyIM {

}
