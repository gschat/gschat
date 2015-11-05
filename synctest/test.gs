package com.gschat;

using com.gschat.Mail;
using gslang.Package;

contract SyncServer {
    void Sync(uint32 num);
}

contract SyncClient {
    @gslang.Async
    void Push(Mail mail);
}
