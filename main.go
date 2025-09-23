package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Client 表示一个SSE客户端连接
type Client struct {
	id      string
	w       http.ResponseWriter
	flusher http.Flusher
}

// BroadcastManager 广播管理器
type BroadcastManager struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

var broadcastManager = &BroadcastManager{
	clients: make(map[string]*Client),
}

// AddClient 添加客户端
func (bm *BroadcastManager) AddClient(id string, w http.ResponseWriter) bool {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return false
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.clients[id] = &Client{
		id:      id,
		w:       w,
		flusher: flusher,
	}
	return true
}

// RemoveClient 移除客户端
func (bm *BroadcastManager) RemoveClient(id string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	delete(bm.clients, id)
}

// Broadcast 向所有客户端广播消息
func (bm *BroadcastManager) Broadcast(message string) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	for id, client := range bm.clients {
		_, err := fmt.Fprintf(client.w, "data: %s\n\n", message)
		if err != nil {
			logrus.Warnf("Failed to send to client %s: %v", id, err)
			// 可以在外层统一清理断开的连接
			continue
		}
		client.flusher.Flush()
	}
}

// ClientCount 返回当前客户端数量
func (bm *BroadcastManager) ClientCount() int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return len(bm.clients)
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 生成客户端ID
	clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())

	// 注册客户端
	if !broadcastManager.AddClient(clientID, w) {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		logrus.Error("SSE not supported")
		return
	}

	logrus.Infof("Client connected: %s (total: %d)", clientID, broadcastManager.ClientCount())

	// 监听连接关闭
	ctx := r.Context()

	// 等待连接关闭
	<-ctx.Done()

	// 连接关闭时清理
	broadcastManager.RemoveClient(clientID)
	logrus.Infof("Client disconnected: %s (total: %d)", clientID, broadcastManager.ClientCount())
}

// startBroadcaster 启动广播器（单线程循环）
func startBroadcaster() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			message := t.Format("2006-01-02 15:04:05")
			broadcastManager.Broadcast(message)
			logrus.Debugf("Broadcasted to %d clients: %s", broadcastManager.ClientCount(), message)
		}
	}
}

func main() {
	// 启动广播器（在单独的goroutine中运行）
	go startBroadcaster()

	http.HandleFunc("/sse", sseHandler)
	logrus.Info("SSE server started at :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logrus.Fatalf("Server failed to start: %v", err)
	}
}
