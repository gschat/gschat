package com.gschat.mq;

using gslang.Lang;

@Lang(Name:"golang",Package:"github.com/gschat/gschat/mq")

table QMeta {
    string          Name            ; // user name
    string          FilePath        ; // message queue persistence file
    uint32          ID              ; // Q's block id
    uint32          Capacity        ; // Q capacity
    QCachedMeta[2]  DuoCached       ; // cached
    uint32          CurrentCached   ; // current using cached
}

table QCachedMeta {
    uint32      Offset      ; // file offset
    uint32      cursor      ; // write cursor
    uint32      MaxSID      ; // Max received sequence id
    uint32      MinSID      ; // Min received sequence id
}
