package shutdown

import "sync"

type Callback interface {
	OnShutdown(string) error
}

type Func func(string) error

func (f Func) OnShutdown(shutdownManager string) error {
	return f(shutdownManager)
}

type ErrorHandler interface {
	OnError(err error)
}

type ErrorFunc func(err error)

func (f ErrorFunc) OnError(err error) {
	f(err)
}

type Manager interface {
	GetName() string
	Start(gs GracefulShutdownI) error
	ShutdownStart() error
	ShutdownFinish() error
}

type GracefulShutdownI interface {
	Start(manager Manager)
	ReportError(err error)
	AddCallback(callback Callback)
}

type GracefulShutdown struct {
	callbacks    []Callback
	managers     []Manager
	errorHandler ErrorHandler
}

func (g *GracefulShutdown) Start(manager Manager) {
	g.ReportError(manager.ShutdownStart())
	var wg sync.WaitGroup
	for _, callback := range g.callbacks {
		wg.Add(1)
		go func(callback Callback) {
			defer wg.Done()
			g.ReportError(callback.OnShutdown(manager.GetName()))
		}(callback)
	}
	wg.Wait()

	g.ReportError(manager.ShutdownFinish())
}

func (g *GracefulShutdown) SetErrorHandler(handler ErrorHandler) {
	g.errorHandler = handler
}

func (g *GracefulShutdown) ReportError(err error) {
	if err != nil && g.errorHandler != nil {
		g.errorHandler.OnError(err)
	}
}

func (g *GracefulShutdown) AddCallback(callback Callback) {
	g.callbacks = append(g.callbacks, callback)
}

func New() *GracefulShutdown {
	return &GracefulShutdown{
		callbacks: make([]Callback, 0, 10),
		managers:  make([]Manager, 0, 3),
	}
}

func (g *GracefulShutdown) StartShutdown() error {
	for _, manager := range g.managers {
		if err := manager.Start(g); err != nil {
			return err
		}
	}
	return nil
}
