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
	"strings"
	"time"
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

	file, err := tail.OpenFile(logFile)
	if err != nil {
		log.Fatal(err)
	}
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	fileLength := int(fileInfo.Size()) // 文件总长度
	foundedLineSeparator := 0          // 已经找到的行数, 每次找到"\n"时加一

	chunkSize := 4096 // 分段读取, 每次读取一些字节, 再遍历这些字节查找"\n"
	if chunkSize > fileLength {
		chunkSize = fileLength
	}
	loopCount := 1                    // 循环次数
	seekPos := -loopCount * chunkSize // 文件指针位置
	for {
		file.Seek(int64(seekPos), io.SeekEnd) // 从末尾开始读取
		byteArr := make([]byte, chunkSize)
		_, err := file.Read(byteArr)
		if err != nil {
			panic(err)
		}
		arrLen := len(byteArr)
		seekPosInChunk := arrLen - 1
		// 遍历读取到的字节
		for ; seekPosInChunk >= 0; seekPosInChunk-- {
			b := byteArr[seekPosInChunk]
			if b == 10 { // 换行符
				foundedLineSeparator++
			}
			if foundedLineSeparator >= tailLineNum { // 找到了, 退出
				break
			}
		}
		seekPos += seekPosInChunk + 1
		if foundedLineSeparator >= tailLineNum { // 找到了, 退出
			break
		}
		loopCount++
		nextSeekPos := -loopCount * chunkSize
		if -nextSeekPos > fileLength {
			// 读取到末尾, 就算找完了
			break
		}
		seekPos = nextSeekPos
	}

	config := tail.Config{
		Follow:    true,
		MustExist: true,
		ReOpen:    true,
		Poll:      true,
		Location: &tail.SeekInfo{
			Offset: int64(seekPos),
			Whence: io.SeekEnd,
		},
	}

	t := tail.Tail{
		Filename: logFile,
		Lines:    make(chan *tail.Line),
		Config:   config,
	}

	t.Logger = log.New(os.Stderr, "", log.LstdFlags)

	watch.POLL_DURATION = 125 * time.Millisecond
	t.Watcher = watch.NewPollingFileWatcher(logFile)

	t.File = file

	go t.TailFileSync()

	reader := bufio.NewReader(os.Stdin)
	go func() {
		for {
			reader.ReadString('\n')
		}
	}()

	var buf strings.Builder
	lineNum := 1
	maxLine := foundedLineSeparator - 1

	// 缓冲文件内容到buf
	for line := range t.Lines {
		if lineNum > maxLine {
			fmt.Println(line.Text)
		} else {
			buf.WriteString(line.Text)
			buf.WriteString("\n")
			if lineNum == maxLine {
				fmt.Print(buf.String())
			}
			lineNum++
		}
	}

}
