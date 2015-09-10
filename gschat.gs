package com.gschat;

using gslang.Exception;
using gslang.Lang;
using gslang.Flag;

@Lang(Name:"golang",Package:"github.com/gschat/gschat")

table Mail {
    uint32      SQID        ; // The IM data's server timestamp
    uint64      TS          ; // IM data timestamp
    string      Sender      ; // IM data sender
    string      Receiver    ; // IM data receiver
    MailType    Type        ; // IM data type {@link DataType}
    string      Content     ; // IM data message

}


enum MailType {
    Single(0),Multi(1),System(2)
}

/**
 * if not found current user {@link IMService} will throw this exception
 */
@Exception
table UserNotFound {}

@Exception
table UserAuthFailed {}


enum ServiceType{
    Unknown,IM,ANPS,Auth,Client
}

table Property{
    string Key;
    string Value;
}


contract IMServer{
    /**
     * put message data into receiver message queue
     * @return the data's service timestamp
     */
    uint64 Put(Mail mail) throws(UserNotFound);
    /**
     *  create new receive stream with newest message's ts of client
     */
    void Pull(uint32 offset) throws(UserNotFound);
}

contract IMAuth{
    Property[] Login(string username,Property[] properties)  throws(UserNotFound,UserAuthFailed);

    void Logoff(Property[] properties);
}

contract IMClient{
    /**
     * push im message to client
     */
    void Push(Mail mail);
    /**
     * notify client newest message timestamp
     */
    void Notify(uint32 SQID);
}
