package manager

import (
	"khetao.com/pkg/shutdown"
	"os"
	"os/signal"
	"syscall"
)

const Name = "PosixSignalManager"

type PosixSignalManager struct {
	signals []os.Signal
}

func (posixSignalManager *PosixSignalManager) ShutdownStart() error {
	return nil
}

func (posixSignalManager *PosixSignalManager) ShutdownFinish() error {
	os.Exit(0)

	return nil
}

func NewPosixSignalManager(sig ...os.Signal) *PosixSignalManager {
	if len(sig) == 0 {
		sig = make([]os.Signal, 2)
		sig[0] = os.Interrupt
		sig[1] = syscall.SIGTERM
	}

	return &PosixSignalManager{
		signals: sig,
	}
}

func (posixSignalManager *PosixSignalManager) GetName() string {
	return Name
}

func (posixSignalManager *PosixSignalManager) Start(gs shutdown.GracefulShutdownI) error {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, posixSignalManager.signals...)

		// Block until a signal is received.
		<-c

		gs.Start(posixSignalManager)
	}()

	return nil
}
