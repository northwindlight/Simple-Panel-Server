package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // 允许跨域

	// 获取Flusher，用于实时推送数据
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		logrus.Error("SSE not supported")
		return
	}

	// 监听连接关闭
	ctx := r.Context()
	ticker := time.NewTicker(2 * time.Second) // 每2秒推送一次
	defer ticker.Stop()

	// for 无限循环
	for {
		// select语句会​​阻塞等待​​直到以下任一case发生
		select {
		case <-ctx.Done():
			logrus.Info("Connection closed by client")
			return
		case t := <-ticker.C:
			// 写入事件数据（格式必须符合SSE规范）
			event := fmt.Sprintf("data: %s\n\n", t.Format("2006-01-02 15:04:05"))
			fmt.Fprint(w, event)
			flusher.Flush() // 立即发送数据到客户端
		}
	}
}

func main() {
	http.HandleFunc("/sse", sseHandler)
	logrus.Info("SSE server started at :8080")
}
