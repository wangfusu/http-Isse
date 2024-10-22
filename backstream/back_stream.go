package backstream

import "sync"

type BSteam[T interface{}] interface {
	Messages() <-chan T
	Stop()
	//OnDone(f func())
}

type DefaultStream[T interface{}] struct {
	InnerMsg    chan T
	stopF       []func()
	doneF       []func()
	stopOnce    sync.Once
	lock        sync.Mutex
	closed      bool
	cacheAllMsg []T
	finally     func(allMsg []T)
	pushTag     bool // 是否推送 全局控制
}

func NewStream[T interface{}]() *DefaultStream[T] {
	return NewStreamByCacheSize[T](32)
}
func NewStreamByCacheSize[T interface{}](cacheSize int) *DefaultStream[T] {
	c := make(chan T, cacheSize)
	return &DefaultStream[T]{InnerMsg: c, cacheAllMsg: make([]T, 0, 10), stopF: make([]func(), 0, 2), doneF: make([]func(), 0, 2), pushTag: true}
}

func NewNotPushStream[T interface{}]() *DefaultStream[T] {
	return &DefaultStream[T]{cacheAllMsg: make([]T, 0, 10), stopF: make([]func(), 0, 2), doneF: make([]func(), 0, 2), pushTag: false}
}

func (d *DefaultStream[T]) OnStop(f func()) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.stopF = append(d.stopF, f)
}

func (d *DefaultStream[T]) Finally(f func(allMsg []T)) {
	d.finally = f
}

func (d *DefaultStream[T]) Done() {
	d.stopOnce.Do(func() {
		d.lock.Lock()
		defer d.lock.Unlock()
		if d.pushTag {
			close(d.InnerMsg)
		}
		d.closed = true
		//if d.doneF != nil {
		//	d.doneF()
		//}
		if d.finally != nil {
			d.finally(d.cacheAllMsg)
		}
	})
}

func (d *DefaultStream[T]) Send(msg T) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if !d.closed {
		if d.pushTag {
			d.InnerMsg <- msg
		}
		d.cacheAllMsg = append(d.cacheAllMsg, msg)
	}
}

func (d *DefaultStream[T]) SendControlPush(msg T, needPush bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if !d.closed {
		if d.pushTag && needPush {
			d.InnerMsg <- msg
		}
		d.cacheAllMsg = append(d.cacheAllMsg, msg)
	}
}

func (d *DefaultStream[T]) Messages() <-chan T {
	return d.InnerMsg
}

func (d *DefaultStream[T]) Stop() {
	d.stopOnce.Do(func() {
		d.lock.Lock()
		defer d.lock.Unlock()
		if d.pushTag {
			close(d.InnerMsg)
		}
		d.closed = true
		if d.stopF != nil && len(d.stopF) > 0 {
			for _, f := range d.stopF {
				f()
			}
		}
		if d.finally != nil {
			d.finally(d.cacheAllMsg)
		}
	})
}
