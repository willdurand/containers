package shim

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
)

func copyStd(s *os.File, w io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()

	reader := bufio.NewReader(s)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		fmt.Fprint(w, line)
	}
}
