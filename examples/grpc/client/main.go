package main

import (
	"context"
	"fmt"
	"log"

	pb "github.com/xingshuo/kite/examples/grpc/proto"
	kite "github.com/xingshuo/kite/pkg"
	"google.golang.org/grpc"
)

var (
	serverUrl = "localhost:5051"
)

type ReqHandler struct {
	conn   *grpc.ClientConn
	client pb.GreeterClient
}

func (rh *ReqHandler) Init(req *kite.Request, results chan<- *kite.Response) error {
	conn, err := grpc.Dial(
		req.Url,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(kite.GRPCClientInterceptor(results, nil)),
	)
	if err != nil {
		return err
	}
	rh.conn = conn
	rh.client = pb.NewGreeterClient(conn)
	return nil
}

func (rh *ReqHandler) OnRequest() error {
	_, err := rh.client.SayHello(context.Background(), &pb.HelloRequest{Name: "lake"})
	return err
}

func (rh *ReqHandler) Close() {
	rh.conn.Close()
}

func main() {
	s := kite.NewServer()
	// test redirect log func
	s.RedirectLog(func(format string, a ...interface{}) (n int, err error) {
		return fmt.Printf("[STAT]:"+format, a...)
	})
	_, err := s.RunWithSimpleArgs(serverUrl, 20, 50, func() kite.ReqHandler {
		return &ReqHandler{}
	})
	if err != nil {
		log.Fatalf("run failed:%v\n", err)
	}
	log.Println("run done")
}
