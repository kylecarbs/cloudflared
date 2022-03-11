package tunnel

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
)

// waitForSignal closes graceShutdownC to indicate that we should start graceful shutdown sequence
func waitForSignal(graceShutdownC chan struct{}, logger *zerolog.Logger) {
	if graceShutdownC == nil {
		return
	}

	signals := make(chan os.Signal, 10)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(signals)

	select {
	case s := <-signals:
		logger.Info().Msgf("Initiating graceful shutdown due to signal %s ...", s)
		select {
		case <-graceShutdownC:
		default:
			close(graceShutdownC)
		}
	case <-graceShutdownC:
	}
}
