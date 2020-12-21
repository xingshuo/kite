del examples\grpc\client\client.exe
del examples\grpc\server\server.exe
del examples\http\client\client.exe
cd examples\grpc\client && go build -gcflags "-N -l"
cd ..\server && go build -gcflags "-N -l"
cd ..\..\http\client && go build -gcflags "-N -l"
cd ..\..\..\