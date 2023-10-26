package main

import (
	"bufio"
	"fmt"
	"github.com/nxadm/tail"
	"io"
	"log"
	"os"
	"strconv"
)

func main() {
	// deal program param
	if len(os.Args) < 2 {
		fmt.Println("need one or more arguments")
		return
	}
	logFile := os.Args[1] // log file path
	tailLineNum := 25     // tail start at last n lines
	if len(os.Args) > 2 {
		tailLineNumArg, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println(os.Args[2] + "is not a int number")
			return
		}
		tailLineNum = tailLineNumArg
	}

	file, err := os.Open(logFile)
	if err != nil {
		log.Fatal(err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	fileLength := fileInfo.Size() // file length

	var seekPos int64
	bufSize := int64(8192)
	if bufSize > fileLength {
		bufSize = fileLength
		seekPos = 0
	} else {
		seekPos = fileLength - bufSize
	}
	foundLines := 0
	for seekPos >= 0 && foundLines < tailLineNum {
		file.Seek(seekPos, io.SeekStart)
		buf := make([]byte, bufSize)
		file.Read(buf)

		for i := bufSize - 1; i >= 0; i-- {
			if buf[i] == 10 {
				foundLines++
			}
			if foundLines == tailLineNum {
				seekPos = seekPos + i
				break
			}
		}
		if foundLines == tailLineNum || seekPos == 0 {
			break
		}

		if seekPos-bufSize < 0 {
			bufSize = seekPos
		}
		seekPos -= bufSize
	}

	// get file length, again
	fileInfo, err = file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	// tail config
	seekInfo := &tail.SeekInfo{
		Offset: seekPos,
		Whence: io.SeekStart,
	}
	config := tail.Config{
		Follow:    true,
		MustExist: true,
		ReOpen:    true,
		Poll:      true,
		Location:  seekInfo,
		Logger:    log.New(os.Stderr, "", log.LstdFlags),
	}

	// watch.POLL_DURATION = 125 * time.Millisecond
	// start tail
	t, _ := tail.TailFile(logFile, config)

	// accept user input \n
	reader := bufio.NewReader(os.Stdin)
	go func() {
		for {
			reader.ReadString('\n')
		}
	}()

	// start print log
	for line := range t.Lines {
		fmt.Println(line.Text)
	}
}
