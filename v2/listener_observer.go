package conntrack

import "net"

type ListenerObserver interface {
	AcceptAttempt(attempt int)
	AcceptFailed(err error)
	ConnectionAccepted(conn net.Conn)
	ConnectionClosed(err error)
}

type ListenerObserverFactory interface {
	NewListenerObserver(name string) ListenerObserver
}

type ListenerObserverFactoryFunc func(name string) ListenerObserver

func (fn ListenerObserverFactoryFunc) NewListenerObserver(name string) ListenerObserver {
	return fn(name)
}

type noOpListenerObserver struct {
	observers []ListenerObserver
}

func NoOpListenerObserver() ListenerObserver {
	return &noOpListenerObserver{}
}

func (o noOpListenerObserver) AcceptAttempt(attempt int)        {}
func (o noOpListenerObserver) AcceptFailed(err error)           {}
func (o noOpListenerObserver) ConnectionAccepted(conn net.Conn) {}
func (o noOpListenerObserver) ConnectionClosed(err error)       {}

type compositeListenerObserver struct {
	observers []ListenerObserver
}

func ListenerObservers(observers ...ListenerObserver) ListenerObserver {
	return &compositeListenerObserver{
		observers: observers,
	}
}

func (co compositeListenerObserver) AcceptAttempt(attempt int) {
	for _, o := range co.observers {
		o.AcceptAttempt(attempt)
	}
}

func (co compositeListenerObserver) AcceptFailed(err error) {
	for _, o := range co.observers {
		o.AcceptFailed(err)
	}
}

func (co compositeListenerObserver) ConnectionAccepted(conn net.Conn) {
	for _, o := range co.observers {
		o.ConnectionAccepted(conn)
	}
}

func (co compositeListenerObserver) ConnectionClosed(err error) {
	for _, o := range co.observers {
		o.ConnectionClosed(err)
	}
}

type compositeListenerObserverFactory struct {
	factories []ListenerObserverFactory
}

func ListenerObserverFactories(factories ...ListenerObserverFactory) ListenerObserverFactory {
	return &compositeListenerObserverFactory{
		factories: factories,
	}
}

func (f compositeListenerObserverFactory) NewListenerObserver(name string) ListenerObserver {
	observers := make([]ListenerObserver, len(f.factories))

	for index, factory := range f.factories {
		observers[index] = factory.NewListenerObserver(name)
	}

	return ListenerObservers(observers...)
}
