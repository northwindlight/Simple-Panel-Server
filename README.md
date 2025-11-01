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

自动创建 config.yaml 和 ./html/。
Windows：以管理员运行（读温度/频率）。
Linux：普通用户即可。
