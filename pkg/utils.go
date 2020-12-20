package kite

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

func GRPCClientInterceptor(results chan<- *Response, filter func(rsp *Response, err error)) grpc.UnaryClientInterceptor {
	var pbMessageInfo proto.InternalMessageInfo
	return func(
		ctx context.Context,
		fullMethod string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		startTime := time.Now()
		err := invoker(ctx, fullMethod, req, resp, cc, opts...)
		rsp := &Response{}
		rsp.UseTime = uint64(time.Since(startTime))
		rsp.Method = fullMethod
		rsp.MsgType = MSG_GRPC
		// 这一部分业务侧可通过filter灵活适配
		if err == nil {
			rsp.IsSucceed = true
			rsp.ErrCode = 0
		} else {
			rsp.IsSucceed = false
			rsp.ErrCode = -1001
		}

		rsp.ReceivedBytes = uint64(pbMessageInfo.Size(resp.(proto.Message)))
		if filter != nil {
			filter(rsp, err)
		}
		results <- rsp
		return err
	}
}

func GenerateHistogram(latencies []float64, slowest, fastest float64) []LatencyBucket {
	bc := 10
	buckets := make([]float64, bc+1)
	counts := make([]int, bc+1)
	bs := (slowest - fastest) / float64(bc)
	for i := 0; i < bc; i++ {
		buckets[i] = fastest + bs*float64(i)
	}
	buckets[bc] = slowest
	var bi int
	for i := 0; i < len(latencies); {
		if latencies[i] <= buckets[bi] {
			i++
			counts[bi]++
		} else if bi < len(buckets)-1 {
			bi++
		}
	}
	res := make([]LatencyBucket, len(buckets))
	for i := 0; i < len(buckets); i++ {
		res[i] = LatencyBucket{
			Mark:      buckets[i],
			Count:     counts[i],
			Frequency: float64(counts[i]) / float64(len(latencies)),
		}
	}
	return res
}

func GenerateLatencies(latencies []float64) []LatencyDistribution {
	if len(latencies) == 0 {
		return nil
	}
	pctls := []int{10, 25, 50, 75, 90, 95, 99}
	data := make([]float64, len(pctls))
	lt := len(latencies)
	for i, p := range pctls {
		fi := float64(p*lt) / float64(100.0)
		ii := int(fi)
		if ii >= lt || float64(ii) == fi {
			ii = ii - 1
		}
		if ii < 0 {
			ii = 0
		}
		data[i] = latencies[ii]
	}

	res := make([]LatencyDistribution, len(pctls))
	for i := 0; i < len(pctls); i++ {
		if data[i] > 0 {
			res[i] = LatencyDistribution{Percentage: pctls[i], Latency: data[i]}
		}
	}
	return res
}
