package com.gschat;

using gslang.Exception;
using gslang.Package;
using gslang.Flag;

@Package(Lang:"golang",Name:"com.gschat",Redirect:"github.com/gschat/gschat")

@Package(Lang:"objc",Name:"com.gschat",Redirect:"GSChat")

table Mail {
    string          MailID      ; // IM data uuid
    uint32          SQID        ; // The IM data's server timestamp
    uint64          TS          ; // IM data timestamp
    string          Sender      ; // IM data sender
    string          Receiver    ; // IM data receiver
    MailType        Type        ; // IM data type {@link DataType}
    string          Content     ; // IM data message
    Attachment[]    Attachments ; // IM attachment list
    byte[]          Extension   ; //
}

table Attachment {
    AttachmentType  Type        ; // attachment type
    byte[]          Content     ; // attachment content
}

enum AttachmentType {
    Text,Image,Video,Audio,GPS,CMD,Customer
}


table AttachmentText {
    string          Text        ; // text content
}

table AttachmentGPS {
    float64         Longitude   ;
    float64         Latitude    ;
    string          Address     ;
}

table AttachmentImage {
    string          Key         ;
    string          Name        ;
}

table AttachmentVideo {
    string          Key         ;
    string          Name        ;
    int16           Duration    ;
}

table AttachmentAudio {
    string          Key         ;
    string          Name        ;
    int16           Duration    ;
}

table AttachmentCMD {
    string          Command     ;
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

@Exception
table ResourceNotFound{}

@Exception
table UnexpectSQID{}

enum ServiceType{
    Unknown,IM,Push,Auth,Client,Status
}

table Property{
    string Key;
    string Value;
}


contract IMServer{
    /**
     * get send SQID
     */
    uint32 Prepare();
    /**
     * put message data into receiver message queue
     * @return the data's service timestamp
     */
    uint64 Put(Mail mail) throws(UserNotFound,UnexpectSQID);

    /**
     *  create new receive stream with newest message's ts of client
     */
    void Pull(uint32 offset) throws(UserNotFound);
}

contract IMAuth{
    Property[] Login(string username,Property[] properties)  throws(UserNotFound,UserAuthFailed);

    void Logoff(Property[] properties);
}

contract IMPush {
    void Register(byte[] pushToken);
    void Unregister();
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
