package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// 配置结构体（简化 WebServer：移除 host/port）
type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
	WebServer struct {
		Enabled   bool   `yaml:"enabled"`
		HTMLDir   string `yaml:"html_dir"`
		IndexFile string `yaml:"index_file"`
	} `yaml:"web_server"`
}

// 全局配置变量
var config Config

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

// Broadcast 向所有客户端广播消息（加 event: update）
func (bm *BroadcastManager) Broadcast(message string) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	for id, client := range bm.clients {
		_, err := fmt.Fprintf(client.w, "event: update\ndata: %s\n\n", message)
		if err != nil {
			logrus.Warnf("Failed to send to client %s: %v", id, err)
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

// loadConfig 加载配置文件
func loadConfig(configPath string) error {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logrus.Warnf("Config file %s not found, creating default config", configPath)
		return createDefaultConfig(configPath)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}
	logrus.Infof("Config loaded from %s", configPath)
	return nil
}

// createDefaultConfig 创建默认配置文件（WebServer 复用 Server 配置）
func createDefaultConfig(configPath string) error {
	// 设置默认值
	config.Server.Host = "0.0.0.0"
	config.Server.Port = 8080
	config.WebServer.Enabled = true
	config.WebServer.HTMLDir = filepath.Join(filepath.Dir(configPath), "html")
	config.WebServer.IndexFile = "index.html"
	// host/port 已移除，直接用 server 的
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %v", err)
	}
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write default config: %v", err)
	}
	logrus.Infof("Default config created at %s", configPath)

	return nil
}

// startWebServer 启动简易HTTP服务器（复用 Server host/port，但实际用 mux 挂载）
func startWebServer() {
	if !config.WebServer.Enabled {
		logrus.Info("Web server is disabled in config")
		return
	}
	// 检查HTML目录是否存在
	if _, err := os.Stat(config.WebServer.HTMLDir); os.IsNotExist(err) {
		logrus.Warnf("HTML directory %s does not exist, creating it", config.WebServer.HTMLDir)
		if err := os.MkdirAll(config.WebServer.HTMLDir, 0755); err != nil {
			logrus.Errorf("Failed to create HTML directory: %v", err)
			return
		}
	}
	logrus.Infof("Web server enabled, static files from %s", config.WebServer.HTMLDir)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	info, err := GetSystemInfo()
	if err != nil {
		logrus.Errorf("Failed to get system info: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(info); err != nil {
		logrus.Errorf("Failed to encode system info: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// 处理 OPTIONS 预检（CORS）
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
		w.WriteHeader(http.StatusOK)
		return
	}
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

// startBroadcaster 启动广播器
func startBroadcaster() {
	ticker := time.NewTicker(2 * time.Second) // 每2秒广播一次
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// 生成系统状态数据
			status := generateSystemStatus()
			// 格式化为JSON字符串
			message := formatStatusData(status)
			// 记录调试信息
			logrus.Debugf("Generated JSON data: %s", message)
			// 广播数据
			broadcastManager.Broadcast(message)
			logrus.Debugf("Broadcasted to %d clients: %s", broadcastManager.ClientCount(), message)
		case <-shutdown:
			logrus.Info("Shutting down broadcaster")
			return
		}
	}
}

// 全局变量用于服务管理
var (
	server   *http.Server
	shutdown = make(chan struct{})
)

type program struct{}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() {
	// 获取可执行文件目录
	exePath, err := os.Executable()
	if err != nil {
		logrus.Fatalf("Failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "config.yaml")

	// 加载配置文件
	if err := loadConfig(configPath); err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// 确保 HTMLDir 是绝对路径
	if !filepath.IsAbs(config.WebServer.HTMLDir) {
		config.WebServer.HTMLDir = filepath.Join(exeDir, config.WebServer.HTMLDir)
		logrus.Infof("Resolved relative HTMLDir to absolute: %s", config.WebServer.HTMLDir)
	} else {
		logrus.Infof("HTMLDir is already absolute: %s", config.WebServer.HTMLDir)
	}

	// 启动广播器（在单独的goroutine中运行）
	go startBroadcaster()
	// 初始化 WebServer（挂载 mux，无需单独启动）
	startWebServer()
	// 统一 mux 处理 SSE 和静态文件
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", sseHandler)
	mux.HandleFunc("/info", infoHandler)
	if config.WebServer.Enabled {
		fs := http.FileServer(http.Dir(config.WebServer.HTMLDir))
		mux.Handle("/", fs)
	}
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	logrus.Infof("Unified server (SSE + Web + Info) started at %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("Server failed to start: %v", err)
	}
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	close(shutdown)
	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logrus.Errorf("Server shutdown error: %v", err)
		}
	}
	return nil
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
	isAdmin()
	svcConfig := &service.Config{
		Name:        "SystemMonitor",
		DisplayName: "System Monitor Service",
		Description: "Monitors system status and broadcasts via SSE",
	}
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logrus.Fatal(err)
	}
	// 新增：检查命令行参数，支持 install/start/stop 等
	if len(os.Args) > 1 {
		err = service.Control(s, os.Args[1])
		if err != nil {
			logrus.Fatalf("Failed to %s service: %v", os.Args[1], err)
		}
		return // 处理完控制命令后退出
	}
	// 否则正常运行服务
	logger, err := s.Logger(nil)
	if err != nil {
		logrus.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
