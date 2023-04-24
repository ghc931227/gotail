package main

import (
	"bufio"
	"fmt"
	"github.com/icza/backscanner"
	"github.com/nxadm/tail"
	"github.com/nxadm/tail/watch"
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

	scanner := backscanner.New(file, int(fileLength))
	lineNum := 0
	stack := &Stack{}
	for lineNum <= tailLineNum {
		line, _, err := scanner.Line()
		if err == nil {
			for i := range line {
				stack.Push(line[i])
			}
			stack.Push(byte(10))
			lineNum++
		} else {
			if err == io.EOF {
				stack.Pop()
			}
			break
		}
	}

	// show checked lines
	checkedBytes := make([]byte, stack.Len())
	for i := range checkedBytes {
		checkedBytes[i] = stack.Pop().(byte)
	}
	fmt.Print(string(checkedBytes[:]))

	// get file length, again
	fileInfo, err = file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	newFileLength := fileInfo.Size()

	// tail config
	var seekInfo *tail.SeekInfo
	if len(checkedBytes) == 0 {
		seekInfo = &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekStart,
		}
	} else {
		seekInfo = &tail.SeekInfo{
			Offset: fileLength - newFileLength - 1,
			Whence: io.SeekEnd,
		}
	}
	config := tail.Config{
		Follow:    true,
		MustExist: true,
		ReOpen:    true,
		Poll:      true,
		Location:  seekInfo,
	}

	t := tail.Tail{
		Filename: logFile,
		Lines:    make(chan *tail.Line),
		Config:   config,
	}

	t.Logger = log.New(os.Stderr, "", log.LstdFlags)
	// watch.POLL_DURATION = 125 * time.Millisecond
	t.Watcher = watch.NewPollingFileWatcher(logFile)
	t.File = file
	// start tail
	go t.TailFileSync()

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

type (
	Stack struct {
		top    *node
		length int
	}
	node struct {
		value interface{}
		prev  *node
	}
)

// Create a new stack
func NewStack() *Stack {
	return &Stack{nil, 0}
}

// Return the number of items in the stack
func (this *Stack) Len() int {
	return this.length
}

// View the top item on the stack
func (this *Stack) Peek() interface{} {
	if this.length == 0 {
		return nil
	}
	return this.top.value
}

// Pop the top item of the stack and return it
func (this *Stack) Pop() interface{} {
	if this.length == 0 {
		return nil
	}
	n := this.top
	this.top = n.prev
	this.length--
	return n.value
}

// Push a value onto the top of the stack
func (this *Stack) Push(value interface{}) {
	n := &node{value, this.top}
	this.top = n
	this.length++
}
