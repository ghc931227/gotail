package main

import (
	"bufio"
	"fmt"
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
	logFile := os.Args[1] // 日志文件路径
	tailLineNum := 25     // 从倒数第几行开始输出日志内容
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
	fileLength := fileInfo.Size() // 文件总长度
	foundedLineSeparator := 0     // 已经找到的行数, 每次找到"\n"时加一

	chunkSize := int64(4096) // 分段读取, 每次读取一些字节, 再遍历这些字节查找"\n"
	if chunkSize > fileLength {
		chunkSize = fileLength
	}
	chunkLoop := int64(1)             // 循环次数
	seekPos := -chunkLoop * chunkSize // 文件指针位置
	stack := &Stack{}
	for foundedLineSeparator < tailLineNum {
		file.Seek(seekPos, io.SeekEnd) // 从末尾开始读取
		byteArr := make([]byte, chunkSize)
		file.Read(byteArr)
		// 倒序遍历读取到的字节
		for seekPosInChunk := chunkSize - 1; seekPosInChunk >= 0 && foundedLineSeparator < tailLineNum; seekPosInChunk-- {
			b := byteArr[seekPosInChunk]
			if b == 10 { // 换行符
				foundedLineSeparator++
			}
			stack.Push(b)
		}
		chunkLoop++
		nextSeekPos := -chunkLoop * chunkSize
		if -nextSeekPos > fileLength {
			// 读取到末尾, 就算找完了
			break
		}
		seekPos = nextSeekPos
	}

	// 遍历过的字节集中显示
	checkedBytes := make([]byte, stack.Len()-1)
	for i := range checkedBytes {
		checkedBytes[i] = stack.Pop().(byte)
	}
	fmt.Print(string(checkedBytes[:]))

	// 重新计算下文件长度
	newFileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	newFileLength := newFileInfo.Size()

	// tail设置
	config := tail.Config{
		Follow:    true,
		MustExist: true,
		ReOpen:    true,
		Poll:      true,
		Location: &tail.SeekInfo{
			// 一般是从-1开始, 就是最后一个字节开始读取, 有些时候在查找行数的时候时间过长,
			// 可能文件长度已经发生了变化, 所以应该从第一次读取文件长度的时候最后一个字节的位置开始
			Offset: fileLength - newFileLength - 1,
			Whence: io.SeekEnd,
		},
	}

	t := tail.Tail{
		Filename: logFile,
		Lines:    make(chan *tail.Line),
		Config:   config,
	}

	t.Logger = log.New(os.Stderr, "", log.LstdFlags)
	//watch.POLL_DURATION = 125 * time.Millisecond
	t.Watcher = watch.NewPollingFileWatcher(logFile)
	t.File = file
	// 开始监视日志滚动
	go t.TailFileSync()

	// 接收用户输入, 比如回车, 以实现console中手动给日志分段
	reader := bufio.NewReader(os.Stdin)
	go func() {
		for {
			reader.ReadString('\n')
		}
	}()

	// 持续输出日志
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
