package kite

import (
	"fmt"
	"sort"
	"time"
)

type Header struct {
	MsgType MsgType
	Method  string
}

type Report struct {
	Header
	TotalUseSec  float64 // 总时长
	ConcyNum     int     // 并行数
	SuccessNum   uint64
	FailureNum   uint64
	QPS          float64
	MaxLatencyMS float64   // 最大延迟
	MinLatencyMS float64   // 最小延迟
	AvgLatencyMS float64   // 平均延迟
	LoadBytes    uint64    // 下载字节数
	LoadSpeed    int64     // 下载速度 bytes/second
	Latencies    []float64 // 升序延迟记录
	Errors       ErrCodes  // 错误码统计
}

func (r *Report) GenerateReport(data *StatisticData) {
	if data.requestTime == 0 {
		data.requestTime = 1
	}
	r.SuccessNum = data.successNum
	r.FailureNum = data.failureNum
	sort.Float64s(data.latencies)
	r.Latencies = data.latencies
	r.QPS = float64(data.successNum*1e9) / float64(data.requestTime)
	if len(r.Latencies) > 0 {
		r.MaxLatencyMS = r.Latencies[len(r.Latencies)-1]
		r.MinLatencyMS = r.Latencies[0]
		sumLatencyMS := float64(0)
		for _, latency := range r.Latencies {
			sumLatencyMS = sumLatencyMS + latency
		}
		// 纳秒=>毫秒
		r.AvgLatencyMS = sumLatencyMS / float64(len(r.Latencies))
	}
	// 纳秒=>秒
	r.TotalUseSec = float64(data.requestTime) / 1e9

	r.LoadBytes = data.receivedBytes
	if r.TotalUseSec > 0 {
		r.LoadSpeed = int64(float64(data.receivedBytes) / r.TotalUseSec)
	}
	r.Errors = data.errors
}

func (r *Report) GenerateHistogram() []LatencyBucket {
	return GenerateHistogram(r.Latencies, r.MaxLatencyMS, r.MinLatencyMS)
}

func (r *Report) GenerateDistribution() []LatencyDistribution {
	return GenerateLatencies(r.Latencies)
}

func (r *Report) OutputReport(logfn LogFunc, logHead string) {
	logfn("%s=======>消息类型|命令字 : %s | %s\n", logHead, r.MsgType, r.Method)
	logfn("─────┬───────┬───────┬───────┬────────┬────────┬────────┬────────┬────────┬────────┬────────\n")
	logfn(" 耗时│ 并发数│ 成功数│ 失败数│   qps  │最长耗时│最短耗时│平均耗时│下载字节│字节每秒│ 错误码\n")
	logfn("─────┼───────┼───────┼───────┼────────┼────────┼────────┼────────┼────────┼────────┼────────\n")
	logfn(fmt.Sprintf("%4.0fs│%7d│%7d│%7d│%8.2f│%6.2fms│%6.2fms│%6.2fms│%8s|%8s│%s\n",
		r.TotalUseSec, r.ConcyNum, r.SuccessNum, r.FailureNum, r.QPS, r.MaxLatencyMS, r.MinLatencyMS, r.AvgLatencyMS,
		fmt.Sprintf("%dB", r.LoadBytes),
		fmt.Sprintf("%dB/s", r.LoadSpeed),
		r.Errors))
	logfn("Latency histogram:\n")
	for _, h := range r.GenerateHistogram() {
		logfn("%8.2fms|%7d|%8.2f%%\n", h.Mark, h.Count, h.Frequency*100)
	}
	logfn("Latency distribution:\n")
	for _, d := range r.GenerateDistribution() {
		logfn("%7d%%     in %8.2fms\n", d.Percentage, d.Latency)
	}
}

type StatisticData struct {
	Header
	logHead       string    // 日志标识
	requestTime   uint64    // 请求总时间
	successNum    uint64    // 成功处理数，code为0
	failureNum    uint64    // 处理失败数，code不为0
	receivedBytes uint64    // 收包量
	latencies     []float64 // 每个包处理时长
	errors        ErrCodes  // 错误码统计
}

type Statistician struct {
	config     *Config
	logfn      LogFunc
	statistics map[Header]*StatisticData
	reports    map[Header]*Report
}

func (s *Statistician) Start(results <-chan *Response, done chan<- []*Report) {
	s.statistics = make(map[Header]*StatisticData)
	s.reports = make(map[Header]*Report)
	logCh := make(chan *StatisticData, 256)
	logDone := make(chan struct{})
	logNo := 0 // 日志流水号，每轮Tick统计自增

	go func() {
		for data := range logCh {
			s.LogReport(data)
		}
		close(logDone)
	}()

	d := time.Duration(1<<63 - 1)
	if s.config.StatFreqSec > 0 {
		d = time.Duration(s.config.StatFreqSec) * time.Second
	}
	ticker := time.NewTicker(d)
	statTime := uint64(time.Now().UnixNano())
	for {
		select {
		case data, actived := <-results:
			// when closed: actived is false, data is nil
			if !actived || data == nil {
				goto exitTag
			}
			header := Header{
				MsgType: data.MsgType,
				Method:  data.Method,
			}
			if s.statistics[header] == nil {
				s.statistics[header] = &StatisticData{
					Header:    header,
					latencies: make([]float64, 0, 256),
					errors:    make(ErrCodes),
				}
			}
			stat := s.statistics[header]
			// 纳秒=>毫秒
			stat.latencies = append(stat.latencies, float64(data.UseTime)/1e6)
			// 是否请求成功
			if data.IsSucceed == true {
				stat.successNum = stat.successNum + 1
			} else {
				stat.failureNum = stat.failureNum + 1
			}
			// 统计错误码
			stat.errors[data.ErrCode] = stat.errors[data.ErrCode] + 1
			// 收包量
			stat.receivedBytes += data.ReceivedBytes
		case <-ticker.C:
			endTime := uint64(time.Now().UnixNano())
			requestTime := endTime - statTime
			logNo++
			for header, stat := range s.statistics {
				lastLatencies := make([]float64, len(stat.latencies))
				copy(lastLatencies, stat.latencies)
				lastErrors := make(ErrCodes, len(stat.errors))
				for errCode, num := range stat.errors {
					lastErrors[errCode] = num
				}
				logCh <- &StatisticData{
					Header:        header,
					logHead:       fmt.Sprintf("[TickNo:%d]", logNo),
					requestTime:   requestTime,
					successNum:    stat.successNum,
					failureNum:    stat.failureNum,
					receivedBytes: stat.receivedBytes,
					latencies:     lastLatencies,
					errors:        lastErrors,
				}
			}
		}
	}

exitTag:
	endTime := uint64(time.Now().UnixNano())
	requestTime := endTime - statTime
	for _, stat := range s.statistics {
		stat.logHead = "[Finally]"
		stat.requestTime = requestTime
		logCh <- stat
	}

	ticker.Stop()
	close(logCh)
	<-logDone
	reports := make([]*Report, 0, len(s.reports))
	for _, r := range s.reports {
		reports = append(reports, r)
	}
	done <- reports
}

func (s *Statistician) LogReport(data *StatisticData) {
	if s.reports[data.Header] == nil {
		s.reports[data.Header] = &Report{
			Header:   data.Header,
			ConcyNum: s.config.ConcurrencyNum,
		}
	}
	report := s.reports[data.Header]
	report.GenerateReport(data)
	report.OutputReport(s.logfn, data.logHead)
}
