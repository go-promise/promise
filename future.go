/**********************************************************\
 *                                                        *
 * promise/future.go                                      *
 *                                                        *
 * future promise implementation for Go.                  *
 *                                                        *
 * LastModified: Oct 2, 2016                              *
 * Author: Ma Bingyao <andot@hprose.com>                  *
 *                                                        *
\**********************************************************/

package promise

import (
	"sync"
	"sync/atomic"
	"time"
)

type subscriber struct {
	onFulfilled OnFulfilled
	onRejected  OnRejected
	next        Promise
}

type future struct {
	value       interface{}
	reason      error
	state       uint32
	subscribers []subscriber
	locker      sync.Mutex
}

// New creates a PENDING Promise object
func New() Promise {
	return new(future)
}

func (p *future) Then(onFulfilled OnFulfilled, rest ...OnRejected) Promise {
	var onRejected OnRejected
	if len(rest) > 0 {
		onRejected = rest[0]
	}
	next := New()
	p.locker.Lock()
	switch State(p.state) {
	case FULFILLED:
		if onFulfilled == nil {
			return &fulfilled{p.value}
		}
		resolve(next, onFulfilled, p.value)
	case REJECTED:
		if onRejected == nil {
			return &rejected{p.reason}
		}
		reject(next, onRejected, p.reason)
	default:
		p.subscribers = append(p.subscribers,
			subscriber{onFulfilled, onRejected, next})
	}
	p.locker.Unlock()
	return next
}

func (p *future) Catch(onRejected OnRejected, test ...func(error) bool) Promise {
	if len(test) == 0 || test[0] == nil {
		return p.Then(nil, onRejected)
	}
	return p.Then(nil, func(e error) (interface{}, error) {
		if test[0](e) {
			return p.Then(nil, onRejected), nil
		}
		return nil, e
	})
}

func (p *future) Complete(onCompleted OnCompleted) Promise {
	return p.Then(onCompleted, onCompleted)
}

func (p *future) WhenComplete(action func()) Promise {
	return p.Then(func(v interface{}) interface{} {
		action()
		return v
	}, func(e error) (interface{}, error) {
		action()
		return nil, e
	})
}

func (p *future) Done(onFulfilled OnFulfilled, onRejected ...OnRejected) {
	p.Then(onFulfilled, onRejected...).Then(nil, func(e error) { go panic(e) })
}

func (p *future) State() State {
	return State(p.state)
}

func (p *future) resolve(value interface{}) {
	p.locker.Lock()
	if atomic.CompareAndSwapUint32(&p.state, uint32(PENDING), uint32(FULFILLED)) {
		p.value = value
		subscribers := p.subscribers
		p.subscribers = nil
		for _, subscriber := range subscribers {
			resolve(subscriber.next, subscriber.onFulfilled, value)
		}
	}
	p.locker.Unlock()
}

func (p *future) Resolve(value interface{}) {
	if promise, ok := value.(*future); ok && promise == p {
		p.Reject(TypeError("Self resolution"))
	} else if promise, ok := value.(Promise); ok {
		promise.Fill(p)
	} else {
		p.resolve(value)
	}
}

func (p *future) Reject(reason error) {
	p.locker.Lock()
	if atomic.CompareAndSwapUint32(&p.state, uint32(PENDING), uint32(REJECTED)) {
		p.reason = reason
		subscribers := p.subscribers
		p.subscribers = nil
		for _, subscriber := range subscribers {
			reject(subscriber.next, subscriber.onRejected, reason)
		}
	}
	p.locker.Unlock()
}

func (p *future) Fill(promise Promise) {
	p.Then(promise.Resolve, promise.Reject)
}

func (p *future) Timeout(duration time.Duration, reason ...error) Promise {
	return timeout(p, duration, reason...)
}

func (p *future) Delay(duration time.Duration) Promise {
	next := New()
	p.Then(func(v interface{}) {
		go func() {
			time.Sleep(duration)
			next.Resolve(v)
		}()
	}, next.Reject)
	return next
}

func (p *future) Tap(onfulfilledSideEffect func(interface{})) Promise {
	return tap(p, onfulfilledSideEffect)
}

func (p *future) Get() (interface{}, error) {
	c := make(chan interface{}, 1)
	p.Then(func(v interface{}) { c <- v }, func(e error) { c <- e })
	v := <-c
	close(c)
	if e, ok := v.(error); ok {
		return nil, e
	}
	return v, nil
}
