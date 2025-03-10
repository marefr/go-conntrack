package conntrack

import (
	"context"
	"net"
)

type DialObserver interface {
	DialAttempt(ctx context.Context, addr string)
	DialFailed(ctx context.Context, err error)
	ConnectionEstablished(ctx context.Context, conn net.Conn)
	ConnectionClosed(ctx context.Context, err error)
}

type DialObserverFactory interface {
	NewDialObserver(name string) DialObserver
}

type DialObserverFactoryFunc func(name string) DialObserver

func (fn DialObserverFactoryFunc) NewDialObserver(name string) DialObserver {
	return fn(name)
}

type noOpDialObserver struct {
	observers []DialObserver
}

func NoOpDialObserver() DialObserver {
	return &noOpDialObserver{}
}

func (o noOpDialObserver) DialAttempt(_ context.Context, _ string)             {}
func (o noOpDialObserver) DialFailed(_ context.Context, _ error)               {}
func (o noOpDialObserver) ConnectionEstablished(_ context.Context, _ net.Conn) {}
func (o noOpDialObserver) ConnectionClosed(_ context.Context, _ error)         {}

type compositeDialObserver struct {
	observers []DialObserver
}

func DialObservers(observers ...DialObserver) DialObserver {
	return &compositeDialObserver{
		observers: observers,
	}
}

func (co *compositeDialObserver) DialAttempt(ctx context.Context, addr string) {
	for _, o := range co.observers {
		o.DialAttempt(ctx, addr)
	}
}

func (co *compositeDialObserver) DialFailed(ctx context.Context, err error) {
	for _, o := range co.observers {
		o.DialFailed(ctx, err)
	}
}

func (co *compositeDialObserver) ConnectionEstablished(ctx context.Context, conn net.Conn) {
	for _, o := range co.observers {
		o.ConnectionEstablished(ctx, conn)
	}
}

func (co *compositeDialObserver) ConnectionClosed(ctx context.Context, err error) {
	for _, o := range co.observers {
		o.ConnectionClosed(ctx, err)
	}
}

type compositeDialObserverFactory struct {
	factories []DialObserverFactory
}

func DialObserverFactories(factories ...DialObserverFactory) DialObserverFactory {
	return &compositeDialObserverFactory{
		factories: factories,
	}
}

func (f compositeDialObserverFactory) NewDialObserver(name string) DialObserver {
	observers := make([]DialObserver, len(f.factories))

	for index, factory := range f.factories {
		observers[index] = factory.NewDialObserver(name)
	}

	return DialObservers(observers...)
}
