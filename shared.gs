package com.gschat;

using gslang.Exception;
using gslang.Package;
using gslang.Flag;
using com.gsrpc.Device;
using com.gsrpc.KV;

table Mail {
    uint64          ID          ; // mail ID
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
table ResourceBusy{}

@Exception
table UnexpectSQID{}

enum Service{
    Unknown(1),MailHub(2),Push(3),Auth(4),Client(5),UserBinder(6),PushServiceProvider(7),DHKeyResolver(8),UserResolverListener(9),UserResolver(10),Gateway
}
