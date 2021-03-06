package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	pb "github.com/xingshuo/kite/examples/grpc/proto"
	kite "github.com/xingshuo/kite/pkg"
	"google.golang.org/grpc"
)

var (
	concyNum       int
	reqNumPerConcy int
	hostUrl        string
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

func init() {
	flag.IntVar(&concyNum, "c", 20, "concurrency num")
	flag.IntVar(&reqNumPerConcy, "n", 50, "per concurrency req num")
	flag.StringVar(&hostUrl, "host", "localhost:5051", "target url")
}

func main() {
	flag.Parse()
	s := kite.NewServer()
	// test redirect log func
	s.RedirectLog(func(format string, a ...interface{}) (n int, err error) {
		return fmt.Printf("[STAT]:"+format, a...)
	})
	_, err := s.RunWithSimpleArgs(hostUrl, concyNum, reqNumPerConcy, func() kite.ReqHandler {
		return &ReqHandler{}
	})
	if err != nil {
		log.Fatalf("run failed:%v\n", err)
	}
	log.Println("run done")
}
