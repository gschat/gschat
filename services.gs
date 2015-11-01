package com.gschat;


using gslang.Exception;
using com.gsrpc.Device;
using com.gschat.UserNotFound;
using com.gschat.ResourceNotFound;
using com.gschat.Mail;

// DH key exchange data
table DHKey {
    string P;
    string G;
}

/**
 * bind user to im service node
 */
contract UserBinder {
    void BindUser(string userid, Device device);
    void UnbindUser(string userid,Device device);
}

/*
 * dhkey resolver service
 */
contract DHKeyResolver {
    DHKey DHKeyResolve(Device device) throws(ResourceNotFound);
}

enum BlockType {
    Discard,Silence
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
    void Push(Mail[] mails);
    void DeviceStatusChanged(Device device,bool online);
    void UserStatusChanged(string userID,Device device, bool online);
    void DeviceRegister(Device device,byte[] token);
    void DeviceUnreigster(Device device);
}
