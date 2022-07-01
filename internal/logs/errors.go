package logs

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

// GetBetterError parses a log file in order to return an error that is more
// informative than the default one passed as a second argument.
func GetBetterError(logFilePath string, defaultError error) error {
	logFile, err := os.Open(logFilePath)
	if err != nil {
		return defaultError
	}
	defer logFile.Close()

	// TODO: do not read the entire file, we only need the more recent lines
	// (last 10 probably).
	data, err := ioutil.ReadAll(logFile)
	if err != nil {
		return defaultError
	}

	// We parse each log line, starting with the most recents first.
	lines := bytes.Split(data, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		log := make(map[string]string)
		if err := json.Unmarshal(lines[i], &log); err != nil {
			continue
		}

		if log["level"] != "error" {
			continue
		}

		msg := log["msg"]
		if msg == "" || msg == "exit status 1" {
			continue
		}

		return errors.New(msg)
	}

	return defaultError
}
