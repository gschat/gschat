package com.gschat;

using gslang.Exception;
using gslang.Lang;

@Lang(Name:"golang",Package:"github.com/gschat/gschat")

table Data {
    uint64      TS          ; // The IM data's server timestamp
    string      Sender      ; // IM data sender
    string      Receiver    ; // IM data receiver
    DataType    Type        ; // IM data type {@link DataType}
    byte[]      Content     ; // IM data content
}


enum DataType {
    Single(0),Multi(1),System(2)
}

/**
 * if not found current user {@link IMService} will throw this exception
 */
@Exception
table UserNotFound {}

@Exception
table UserAuthFailed {}

contract IMServer{
    /**
     * put message data into receiver message queue
     * @return the data's service timestamp
     */
    uint64 Put(Data data) throws(UserNotFound);
    /**
     *  create new receive stream with newest message's ts of client
     */
    void Pull(uint64 ts);
}

contract IMAuth{
    /**
     * login with username and password
     */
    void Login(string username,string password) throws(UserNotFound,UserAuthFailed);
}

contract IMClient{
    /**
     * push im message to client
     */
    void Push(Data data);
    /**
     * notify client newest message timestamp
     */
    void Notify(uint64 ts);
}
