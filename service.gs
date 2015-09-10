package com.gschat;


using gslang.Exception;
using gslang.Lang;
using com.gschat.ServiceType;
using com.gsrpc.Device;

@Lang(Name:"golang",Package:"github.com/gschat/gschat")

table NamedService {
    string          Name;
    ServiceType     Type;
    uint32          VNodes;
}

/**
 * micoservice interface
 */
contract Service {
    /**
     * get service name
     */
    NamedService Name();
}

/**
 * im manager interface
 */
contract IManager {
    /**
     * bind user with device
     */
    void Bind(string username,Device device);
    /**
     * unbind user from device
     */
    void Unbind(string username,Device device);
}
