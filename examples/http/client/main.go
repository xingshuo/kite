package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"strings"
	"time"

	kite "github.com/xingshuo/kite/pkg"
)

var (
	concyNum       int
	reqNumPerConcy int
	hostUrl        string
)

type ReqHandler struct {
	client  *http.Client
	httpReq *http.Request
}

func (rh *ReqHandler) Init(req *kite.Request, results chan<- *kite.Response) error {
	// 跳过证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	rh.client = &http.Client{
		Transport: kite.HTTPClientInterceptor(results, tr, nil),
		Timeout:   5 * time.Second,
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded; charset=utf-8"
	httpReq, err := http.NewRequest("GET", req.Url, strings.NewReader(""))
	if err != nil {
		return err
	}
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
	rh.httpReq = httpReq
	return nil
}

func (rh *ReqHandler) OnRequest() error {
	_, err := rh.client.Do(rh.httpReq)
	return err
}

func (rh *ReqHandler) Close() {
}

func init() {
	flag.IntVar(&concyNum, "c", 20, "concurrency num")
	flag.IntVar(&reqNumPerConcy, "n", 50, "per concurrency req num")
	flag.StringVar(&hostUrl, "host", "https://www.baidu.com", "target url")
}

func main() {
	flag.Parse()
	s := kite.NewServer()
	_, err := s.RunWithSimpleArgs(hostUrl, concyNum, reqNumPerConcy, func() kite.ReqHandler {
		return &ReqHandler{}
	})
	if err != nil {
		log.Fatalf("run failed:%v\n", err)
	}
	log.Println("run done")
}
