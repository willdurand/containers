package log

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type LogFile struct {
	sync.Mutex

	file *os.File
}

func NewFile(name string) (*LogFile, error) {
	file, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	return &LogFile{file: file}, nil
}

func (l *LogFile) WriteMessage(s, m string) {
	l.Lock()
	defer l.Unlock()

	data, err := json.Marshal(map[string]interface{}{
		"t": time.Now().UTC(),
		"m": m,
		"s": s,
	})

	if err == nil {
		if _, err := l.file.Write(append(data, '\n')); err != nil {
			logrus.WithFields(logrus.Fields{
				"s":     s,
				"error": err,
			}).Warn("failed to write to container log file")
		}
	}
}

func (l *LogFile) Close() error {
	return l.file.Close()
}
