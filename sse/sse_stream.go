package sse

import (
	"fmt"
	"http-Isse/util"
	"net/http"
	"sync"
)

type EventData struct {
	ID      []byte //数据标识符用id字段表示，相当于每一条数据的编号。
	Data    []byte //数据内容用data字段表示。如果数据很长，可以分成多行，最后一行用\n\n结尾，前面行都用\n结尾。
	Event   []byte //event字段表示自定义的事件类型，默认是message事件
	Retry   []byte //服务器可以用retry字段，指定浏览器重新发起连接的时间间隔。 retry: 10000\n
	Comment []byte //表示注释。通常，服务器每隔一段时间就会向浏览器发送一个注释，保持连接不中断。
}

type Stream interface {
	SendEventData(edata *EventData) error
	Done()
	Wait()
	startAsyncPush() error
	OnClose(f func())
}

type sseStream struct {
	onClose  func()
	event    chan *EventData
	quit     chan struct{}
	req      *http.Request
	res      http.ResponseWriter
	Headers  map[string]string
	quitOnce sync.Once
	doneOnce sync.Once
	lock     sync.Mutex
	closed   bool
}

func (s *sseStream) OnClose(f func()) {
	s.onClose = f
}

func (s *sseStream) Wait() {
	<-s.quit
}

func CreateStream(req *http.Request, res http.ResponseWriter) (Stream, error) {
	ss := &sseStream{}
	ss.event = make(chan *EventData, 32)
	ss.quit = make(chan struct{})
	ss.req = req
	ss.res = res
	err := ss.startAsyncPush()
	if err != nil {
		return nil, err
	}
	return ss, nil
}

func (s *sseStream) SendEventData(edata *EventData) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = util.HandleRecoverError(r)
		}
	}()
	s.lock.Lock()
	defer s.lock.Unlock()
	if !s.closed {
		s.event <- edata
	}
	return nil
}

func (s *sseStream) startAsyncPush() (oerr error) {
	defer func() {
		if r := recover(); r != nil {
			oerr = util.HandleRecoverError(r)
		}
	}()
	flusher, err := (s.res).(http.Flusher)
	if !err {
		return fmt.Errorf("streaming unsupported")
	}
	(s.res).Header().Set("Content-Type", "text/event-stream")
	(s.res).Header().Set("Cache-Control", "no-cache")
	(s.res).Header().Set("Connection", "keep-alive")
	(s.res).Header().Set("Transfer-Encoding", "chunked")
	//(s.res).Header().Set("Content-Encoding", "gzip")

	for k, v := range s.Headers {
		(s.res).Header().Set(k, v)
	}
	go func() {
		<-s.req.Context().Done()
		s.close()
	}()

	(s.res).WriteHeader(http.StatusOK)
	flusher.Flush()

	go func(str *sseStream) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println(util.HandleRecoverError(r))
			}
		}()
		for {
			select {
			// Publish event to subscribers
			case ev, ok := <-str.event:
				if !ok {
					str.sendDoneQuit()
					break
				}
				if len(ev.Data) == 0 && len(ev.Comment) == 0 {
					break
				}
				if len(ev.Data) > 0 {
					_, _ = fmt.Fprintf(str.res, "id: %s\n", ev.ID)
					_, _ = fmt.Fprintf(str.res, "data: %s\n", ev.Data)
					if len(ev.Event) > 0 {
						_, _ = fmt.Fprintf(str.res, "event: %s\n", ev.Event)
					}
					if len(ev.Retry) > 0 {
						_, _ = fmt.Fprintf(str.res, "retry: %s\n", ev.Retry)
					}
				}
				if len(ev.Comment) > 0 {
					_, _ = fmt.Fprintf(str.res, ": %s\n", ev.Comment)
				}
				_, _ = fmt.Fprint(str.res, "\n")
				flusher.Flush()

			// Shutdown if the server closes
			case <-str.quit:
				return
			}
		}
	}(s)
	return nil
}

func (s *sseStream) Done() {
	s.doneOnce.Do(func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		close(s.event)
		s.closed = true
	})
}

func (s *sseStream) sendDoneQuit() {
	s.quitOnce.Do(func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		close(s.quit)
		s.closed = true
	})
}

func (s *sseStream) close() {
	s.quitOnce.Do(func() {
		s.lock.Lock()
		defer s.lock.Unlock()
		close(s.quit)
		s.closed = true
		if s.onClose != nil {
			s.onClose()
		}
	})
}
