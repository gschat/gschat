package com.gschat.eventQ;

using com.gsrpc.Device;
using gslang.Package;

@Package(Lang:"golang",Name:"com.gschat.eventQ",Redirect:"github.com/gschat/gschat/eventQ")

table DeviceStatus {
    Device       Device;
    bool         Online;
}


table UserStatus {
    string       Name;
    Device       Device;
    bool         Online;
}


table PushToken {
    Device       Device;
    string       Token;
    bool         Bind;
}
