// Package logger fornece o logger estruturado (JSON) usado pela aplicação.
package logger

import (
	"log/slog"
	"os"
)

// New cria um logger JSON para stdout. Em produção o nível é Info;
// fora de produção, Debug.
func New(appEnv string) *slog.Logger {
	level := slog.LevelDebug
	if appEnv == "production" {
		level = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(h)
}
