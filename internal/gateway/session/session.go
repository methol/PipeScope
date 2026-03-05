package session

import "time"

type Event struct {
	RuleID     string
	ListenPort int
	SrcAddr    string
	DstAddr    string
	StartTS    int64
	EndTS      int64
	DurationMS int64
	UpBytes    int64
	DownBytes  int64
	TotalBytes int64
	Status     string
	Error      string
}

type ConnSession struct {
	RuleID     string
	ListenPort int
	SrcAddr    string
	DstAddr    string
	StartTS    int64
	EndTS      int64
	UpBytes    int64
	DownBytes  int64
	Status     string
	Error      string
}

func New(ruleID string, listenPort int, srcAddr, dstAddr string) *ConnSession {
	return &ConnSession{
		RuleID:     ruleID,
		ListenPort: listenPort,
		SrcAddr:    srcAddr,
		DstAddr:    dstAddr,
		StartTS:    time.Now().UnixMilli(),
		Status:     "ok",
	}
}

func (s *ConnSession) AddUpBytes(n int64) {
	s.UpBytes += n
}

func (s *ConnSession) AddDownBytes(n int64) {
	s.DownBytes += n
}

func (s *ConnSession) MarkDialFail(err error) {
	s.Status = "dial_fail"
	if err != nil {
		s.Error = err.Error()
	}
}

func (s *ConnSession) MarkTimeout(err error) {
	s.Status = "timeout"
	if err != nil {
		s.Error = err.Error()
	}
}

func (s *ConnSession) MarkIOErr(err error) {
	s.Status = "io_err"
	if err != nil {
		s.Error = err.Error()
	}
}

func (s *ConnSession) Finalize() Event {
	endTS := s.EndTS
	if endTS == 0 {
		endTS = time.Now().UnixMilli()
	}
	dur := endTS - s.StartTS
	if dur < 0 {
		dur = 0
	}
	total := s.UpBytes + s.DownBytes
	return Event{
		RuleID:     s.RuleID,
		ListenPort: s.ListenPort,
		SrcAddr:    s.SrcAddr,
		DstAddr:    s.DstAddr,
		StartTS:    s.StartTS,
		EndTS:      endTS,
		DurationMS: dur,
		UpBytes:    s.UpBytes,
		DownBytes:  s.DownBytes,
		TotalBytes: total,
		Status:     s.Status,
		Error:      s.Error,
	}
}

