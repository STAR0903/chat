package main

import (
	"chat/pkg"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func main() {
	// 1.tcp连接硬件
	listener, err := net.Listen("tcp", "0.0.0.0:8005")
	if err != nil {
		log.Println("listen error:", err)
		return
	}
	log.Println("listening")
	defer listener.Close()
	// 接受客户端连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Serve connect error:", err)
			continue
		}
		log.Println("Serve Connect")
		go func(conn net.Conn) {

			var wg sync.WaitGroup
			var IsDone = make(chan bool)
			var IsRead = make(chan bool)
			wg.Add(3)

			go func() {
				defer wg.Done()
				Server(conn, IsRead)
			}()

			// ai语音对话
			go func() {
				defer wg.Done()
				handle(IsRead, IsDone)
			}()

			// tcp传递给硬件
			go func() {
				defer wg.Done()
				Client(conn, IsDone)
			}()
			wg.Wait()
			conn.Close()
		}(conn)
	}

}

// ai语音对话
func handle(IsRead, IsDone chan bool) {
	// 一、语音识别
	// 1. 创建语音识别任务
	<-IsRead
	taskId, err := pkg.CreateRecTask()
	if err != nil {
		log.Printf("Failed to create recognition task: %v\n", err)
		return
	}
	log.Printf("Recognition task created successfully, TaskId: %d\n", *taskId)

	// 2. 轮询查询任务状态（异步任务需要等待执行完成）
	var data string                 // 语音识别数据
	pollInterval := 2 * time.Second // 轮询间隔（可根据音频长度调整，建议2-5秒）
	maxRetries := 30                // 最大轮询次数（避免无限等待，超时可适当增加）
	retryCount := 0

	for {
		if retryCount >= maxRetries {
			log.Println("Polling timed out. The task may still be processing.")
			return
		}

		// 查询当前任务状态
		status, result, err := pkg.DescribeTaskStatus(taskId)
		if err != nil {
			log.Printf("Failed to query task status: %v\n", err)
			break
		}

		// 根据状态码处理不同情况（状态码含义参考腾讯云API文档）
		switch *status {
		case 0:
			fmt.Println("Task status: Waiting. Continuing to query...")
		case 1:
			fmt.Println("Task status: Doing. Continuing to query...")
		case 2:
			fmt.Printf("Task completed successfully! Recognition result:\n%s\n", *result)
			data = (*result)[19:]
		case 3:
			fmt.Printf("Task failed: %v\n", err)
		}

		if *status >= 2 {
			break
		}

		// 等待指定间隔后再次查询
		time.Sleep(pollInterval)
		retryCount++

	}

	// 二、ai生成回答
	answer, err := pkg.Chat(data)
	if err != nil {
		log.Printf("Failed to chat: %v\n", err)
		return
	} else {
		log.Printf("Chat completed successfully! The answer is:\n%s\n", answer)
	}

	// 三、语音合成
	err = pkg.TextToVoice(answer)
	if err != nil {
		log.Printf("Failed to TextToVoice: %v\n", err)
		return
	} else {
		log.Printf("语音合成成功！\n")
	}

	IsDone <- true
}

// 接收
func Server(conn net.Conn, IsRead chan bool) {
	// 创建新文件
	fileName := "data/input.wav"
	f, err := os.Create(fileName)
	if err != nil {
		log.Println("create file err:", err)
		return
	}
	defer f.Close()
	// 接收客户端发送文件内容，原封不动写入文件
	buf := make([]byte, 4096)
	for {
		conn.SetReadDeadline(time.Now().Add(20 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF || n == 0 {
				log.Println("read success")
				IsRead <- true
				return
			} else {
				fmt.Println("read err:", err)
				return
			}
		}
		_, err = f.Write(buf[:n]) // 写入文件，读多少写多少
		if err != nil {
			fmt.Println("write err:", err)
			return
		}
	}
}

// 发送
func Client(conn net.Conn, IsDone chan bool) {

	<-IsDone
	data, err := os.ReadFile("data/output.wav")
	if err != nil {
		log.Println("read file error:", err)
	}
	_, err = conn.Write(data)
	if err != nil {
		log.Println("write error:", err)
	}
	log.Println("write success")

}
