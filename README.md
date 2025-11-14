Simple Panel Server
一个 Go + SSE 的实时系统监控面板。跨 Linux/Windows，监控 CPU/内存/磁盘/温度/频率，前端浏览器仪表盘。

克隆 & 构建：
```shell
clone https://github.com/northwindlight/Simple-Panel-Server.git
cd Simple-Panel-Server
go mod download
go build -o panel .
```

运行：
```shell
./panel
```

自动创建 config.yaml。
Windows：以管理员运行（读温度/频率）。
Linux：普通用户即可。

作为服务运行
以管理员/root 权限执行：
```shell
# 安装
./panel install

# 启动/停止/重启
./panel start
./panel stop
./panel restart

# 卸载
./panel uninstall

# 状态
# Windows: sc query SystemMonitor
# Linux: systemctl status systemmonitor
```

配置
编辑 config.yaml：
```yaml
server:
  host: "0.0.0.0"
  port: 8080
web_server:
  enabled: true
  html_dir: "./html"
  index_file: "index.html"
  ```
