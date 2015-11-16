package com.gschat;

using gslang.Exception;
using gslang.Package;
using gslang.Flag;
using com.gsrpc.Device;
using com.gsrpc.KV;
using com.gschat.Mail;
using com.gschat.UserNotFound;
using com.gschat.UnexpectSQID;
using com.gschat.ResourceNotFound;
using com.gschat.ResourceBusy;
using com.gschat.UserAuthFailed;

@Package(Lang:"golang",Name:"com.gschat",Redirect:"github.com/gschat/gschat")

@Package(Lang:"objc",Name:"com.gschat",Redirect:"GSChat")

table Sync {
    uint32      Offset;
    uint32      Count;
}

contract MailHub{
    /**
     * get put SQID
     */
    uint32 PutSync();
    /**
     * put message data into receiver message queue
     * @return the data's service timestamp
     */
    uint64 Put(Mail mail) throws(UserNotFound,UnexpectSQID);

    /**
     *  sync messages
     */
    Sync Sync(uint32 offset,uint32 count) throws(UserNotFound,ResourceNotFound,ResourceBusy);

    /**
     * finish one sync stream with last message id
     */
    void Fin(uint32 offset);

    /**
     * fetch mails by mail SQID list
     */
    Mail[] Fetch(uint32[] mailIDs) throws(ResourceNotFound);
}

contract Auth{
    KV[] Login(string username,KV[] properties)  throws(UserNotFound,UserAuthFailed,ResourceNotFound);

    void Logoff(KV[] properties);
}

contract Push {
    void Register(byte[] pushToken);
    void Unregister();
}

contract Client{
    /**
     * push im message to client
     */
    @gslang.Async
    void Push(Mail mail);

    /**
     * notify client the newest message's SQID
     */
    @gslang.Async
    void Notify(uint32 SQID);

    /**
     * notify client other device with which the same user login state changed event
     */
     @gslang.Async
    void DeviceStateChanged(Device device,bool online);
}
