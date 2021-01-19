package kite

import (
	"fmt"
	"sync"
)

type Server struct {
	logfn LogFunc
}

func (s *Server) init() {
	s.logfn = fmt.Printf
}

func (s *Server) newTransport(cfg *Config, req *Request, results chan<- *Response, handler ReqHandler) error {
	err := handler.Init(req, results)
	if err != nil {
		return err
	}
	for i := 0; i < cfg.ReqNumPerConcy; i++ {
		err = handler.OnRequest()
		if err != nil {
			fmt.Printf("on request %s err:%v\n", req.Url, err)
		}
	}
	handler.Close()
	return nil
}

func (s *Server) Run(cfg *Config, req *Request, newHandler NewReqHandlerFunc) ([]*Report, error) {
	var wg sync.WaitGroup
	results := make(chan *Response, cfg.ResultsBufferSize)
	done := make(chan []*Report)
	stat := &Statistician{config: cfg, logfn: s.logfn}
	go stat.Start(results, done)
	for i := 0; i < cfg.ConcurrencyNum; i++ {
		wg.Add(1)
		go func() {
			err := s.newTransport(cfg, req, results, newHandler())
			if err != nil {
				fmt.Printf("new transport err:%v", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	close(results)
	reports := <-done
	return reports, nil
}

func (s *Server) RunWithSimpleArgs(targetUrl string, concyNum int, reqNumPerConcy int, newHandler NewReqHandlerFunc) ([]*Report, error) {
	cfg := &Config{
		ConcurrencyNum:    concyNum,
		StatFreqSec:       0, // 默认不会定期输出
		ResultsBufferSize: 1024,
		ReqNumPerConcy:    reqNumPerConcy,
	}
	req := &Request{Url: targetUrl}
	return s.Run(cfg, req, newHandler)
}

func (s *Server) RedirectLog(logfn LogFunc) {
	if logfn != nil {
		s.logfn = logfn
	}
}
