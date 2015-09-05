package com.gschat;


using gslang.Exception;
using gslang.Lang;

@Lang(Name:"golang",Package:"github.com/gschat/gschat")

enum ServiceType{
    IM,ANPS,Auth
}

/**
 * micoservice interface
 */
contract Service {
    /**
     * get service type
     */
    ServiceType Type();
}


/**
 * implement IM service
 */
contract IMService {
    /**
     * get im service name,which will be used as consistent hash key
     */
    string Name();

    /**
     * virtual nodes
     */
    uint32 VNodes();
}
