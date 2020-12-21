Kite
========
    压测工具
    目前以拦截器的方式支持grpc, http

Usage
-----
    参考examples下用例实现自己业务的ReqHandler
    `
    type ReqHandler interface {
    	Init(req *Request, results chan<- *Response) error
    	OnRequest() error
    	Close()
    }
    `
Build
-----
    windows:
         .\build.bat
    linux:
         运行`sh generate_linux_shell.sh`生成对应的build.sh
         .\build.sh