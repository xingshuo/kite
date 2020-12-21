package kite

import (
	"fmt"
	"sort"
	"strings"
)

type ReqHandler interface {
	Init(req *Request, results chan<- *Response) error
	OnRequest() error
	Close()
}

type NewReqHandlerFunc func() ReqHandler

type LogFunc func(format string, a ...interface{}) (n int, err error)

type Config struct {
	ConcurrencyNum    int
	StatFreqSec       int
	ResultsBufferSize int
	ReqNumPerConcy    int
}

// 请求内容
type Request struct {
	Url string
}

// 请求结果
type Response struct {
	MsgType       MsgType
	Method        string
	UseTime       uint64 // 请求时间 纳秒
	IsSucceed     bool   // 是否请求成功
	ErrCode       int    // 错误码
	ReceivedBytes uint64
}

type MsgType int

const (
	MSG_GRPC MsgType = 1
	MSG_MQ   MsgType = 2
	MSG_HTTP MsgType = 3
)

func (mt MsgType) String() string {
	switch mt {
	case MSG_GRPC:
		return "grpc"
	case MSG_MQ:
		return "mq"
	case MSG_HTTP:
		return "http"
	default:
		return "unknown"
	}
}

type LatencyBucket struct {
	// The Mark for histogram bucket in milliseconds
	Mark float64 `json:"mark"`

	// The count in the bucket
	Count int `json:"count"`

	// The frequency of results in the bucket as a decimal percentage
	Frequency float64 `json:"frequency"`
}

// LatencyDistribution holds latency distribution data
type LatencyDistribution struct {
	Percentage int `json:"percentage"`
	// The Mark for distribution in milliseconds
	Latency float64 `json:"latency"`
}

// 统计错误码
type ErrCodes map[int]int

func (e ErrCodes) String() string {
	list := make([]string, 0, len(e))
	for k, v := range e {
		list = append(list, fmt.Sprintf("%d:%d", k, v))
	}
	sort.Strings(list)
	return strings.Join(list, ";")
}
