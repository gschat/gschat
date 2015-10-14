package com.gschat;


using gslang.Exception;
using com.gschat.ServiceType;
using com.gsrpc.Device;
using com.gschat.UserNotFound;
using com.gschat.ResourceNotFound;

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

table DHKey {
    string P;
    string G;
}

/*
 * dhkey resolver service
 */
contract DHKeyResolver {
    DHKey DHKeyResolve(Device device) throws(ResourceNotFound);
}

enum BlockType {
    Discard,Slience
}

table BlockRule {
    string      Target;
    BlockType   BlockType;
}

/*
 * user system service
 */
contract UserResolver {
    // get group user list
    string[] QueryGroup(string groupID) throws(ResourceNotFound);
    // get user's block list
    BlockRule[] QueryBlockRule(string userID);
}

/*
 * user info listener
 */
contract UserResolverListener {
    void GroupChanged(string groupID, string[] added, string[] removed);
    void GroupRemoved(string groupID);
    void BlockRuleChanged(string userID,BlockRule[] added, BlockRule[] removed);
}

contract PushServiceProvider {
    void DeviceStatusChanged(Device device,bool online);
    void UserStatusChanged(string userID,Device device, bool online);
    void DeviceRegister(Device device,byte[] token);
    void DeviceUnreigster(Device device);
}
