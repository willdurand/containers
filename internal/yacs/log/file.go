package log

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type LogFile struct {
	file *os.File
	sync.Mutex
}

func NewFile(name string) (*LogFile, error) {
	file, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	return &LogFile{file: file}, nil
}

func (l *LogFile) Write(p []byte) (int, error) {
	l.Lock()
	defer l.Unlock()
	return l.file.Write(p)
}

func (l *LogFile) Close() error {
	return l.file.Close()
}

func (l *LogFile) WriteStream(r io.Reader, name string) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		data, err := json.Marshal(map[string]interface{}{
			"t": time.Now().UTC(),
			"m": scanner.Text(),
			"s": name,
		})

		if err == nil {
			if _, err := l.Write(append(data, '\n')); err != nil {
				logrus.WithFields(logrus.Fields{
					"s":     name,
					"error": err,
				}).Warn("failed to write to container log file")
			}
		}
	}
}
