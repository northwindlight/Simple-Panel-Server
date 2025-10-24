// Updated script.js (only SystemInfoManager section modified; rest unchanged)
// MonitorSystem - Modular refactor: Connection/Data/UI managers, fixed bugs (URLs, adaptation, duplicates, thresholds, tempDesc)
const MonitorSystem = (() => {
    // Constants
    const UNITS = {
        cpu: { suffix: '<small>%</small>', max: 100, factor: 3.6, threshold: { high: 80, danger: 90 } },
        temp: { suffix: '<small>℃</small>', max: 100, factor: 3.6, threshold: { high: 65, danger: 75 } },
        mem: { suffix: '<small>%</small>', max: 100, factor: 3.6, threshold: { high: 80, danger: 90 } },
        disk: { suffix: '<small>%</small>', max: 100, factor: 3.6, threshold: { high: 80, danger: 90 } }
    };

    const SSE_ENDPOINT = 'http://127.0.0.1:8080/sse'; // Fixed: local
    const INFO_ENDPOINT = 'http://127.0.0.1:8080/info';
    const CHART_POINTS = 20;
    const ANIMATION_FRAMES = 60;

    // DOM Cache
    const elements = {};
    const infoElements = {};
    function cacheElements() {
        const domMap = {
            cpu: 'cpu', temp: 'temp', mem: 'mem', disk: 'disk',
            cpuParams: 'cpuParams', tempParams: 'tempParams', memParams: 'memParams', diskParams: 'diskParams',
            cpuAngle: 'cpuAngle', tempAngle: 'tempAngle', memAngle: 'memAngle', diskAngle: 'diskAngle',
            processorFreq: 'processorFreq', tempDesc: 'tempDesc', memSpace: 'memSpace', diskSpace: 'diskSpace',
            connectionStatus: 'connectionStatus', connectionText: 'connectionText',
            processorStatus: 'processorStatus', tempStatus: 'tempStatus', memStatus: 'memStatus', diskStatus: 'diskStatus',
            cpuChart: 'cpu-chart', tempChart: 'temp-chart', memChart: 'mem-chart', diskChart: 'disk-chart',
            refreshBtn: 'refreshBtn'
        };
        Object.entries(domMap).forEach(([key, id]) => elements[key] = document.getElementById(id));

        const infoMap = {
            sysName: 'sys-name', sysUptime: 'sys-uptime', sysKernel: 'sys-kernel',
            sysCpu: 'sys-cpu', sysSoc: 'sys-soc', sysMem: 'sys-mem', sysDisk: 'sys-disk'
        };
        Object.entries(infoMap).forEach(([key, id]) => infoElements[key] = document.getElementById(id));
    }

    // State
    const state = {
        data: { cpu: 0, temp: 0, mem: 0, disk: 0 },
        current: { cpu: 0, temp: 0, mem: 0, disk: 0 },
        steps: { cpu: 0, temp: 0, mem: 0, disk: 0 },
        source: null,
        chartData: { cpu: Array(CHART_POINTS).fill(0), temp: Array(CHART_POINTS).fill(0), mem: Array(CHART_POINTS).fill(0), disk: Array(CHART_POINTS).fill(0) }
    };

    const systemInfo = {
        startTime: Date.now(), // Will be set based on fetched uptime
        uptimeInterval: null
    };

    // Data Adapter
    class DataAdapter {
        static adapt(rawData) {
            if (!rawData) return null;
            return {
                cpu: rawData.cpu_usage || 0,
                temp: rawData.temperature || 0,
                mem: rawData.memory_usage || 0,
                disk: rawData.storage_usage || 0,
                freq: rawData.cpu_frequency || 0,
                mem_used: rawData.memory_used || 0,
                mem_total: rawData.memory_total || 0,
                disk_used: rawData.storage_used || 0,
                disk_total: rawData.storage_total || 0
            };
        }

        static getStatusConfig(key, value) {
            const thresholds = UNITS[key].threshold;
            if (value > thresholds.danger) return { text: '危险', class: 'danger' };
            if (value > thresholds.high) {
                let text = '高负载';
                if (key === 'temp') text = '高温';
                else if (key === 'mem' || key === 'disk') text = '高使用率';
                return { text, class: 'high' };
            }
            return { text: '正常', class: 'normal' };
        }
    }

    // Connection Manager
    class ConnectionManager {
        constructor() { this.source = null; }

        connect() {
            if (this.source) this.source.close();
            this.source = new EventSource(SSE_ENDPOINT);

            this.source.onopen = () => UIManager.setConnectionStatus(true);

            this.source.addEventListener('update', (event) => {
                try {
                    const rawData = JSON.parse(event.data);
                    const adapted = DataAdapter.adapt(rawData);
                    if (adapted) StateManager.updateState(adapted);
                } catch (e) {
                    console.error("Data parsing error:", e);
                }
            });

            this.source.onerror = () => {
                UIManager.setConnectionStatus(false);
                this.source.close();
                setTimeout(() => this.connect(), 3000);
            };
        }

        disconnect() { if (this.source) this.source.close(); }
    }

    // State Manager
    const StateManager = {
        updateState(adapted) {
            state.data.cpu = adapted.cpu;
            state.data.temp = adapted.temp;
            state.data.mem = adapted.mem;
            state.data.disk = adapted.disk;

            // Update chart data (fixed: add historical shift/push)
            state.chartData.cpu.shift(); state.chartData.cpu.push(adapted.cpu);
            state.chartData.temp.shift(); state.chartData.temp.push(adapted.temp);
            state.chartData.mem.shift(); state.chartData.mem.push(adapted.mem);
            state.chartData.disk.shift(); state.chartData.disk.push(adapted.disk);

            // Update text (direct backend units)
            if (elements.processorFreq) elements.processorFreq.textContent = `${adapted.freq} MHz`;
            if (elements.memSpace) elements.memSpace.textContent = `${adapted.mem_used} MB / ${adapted.mem_total} MB`;
            if (elements.diskSpace) elements.diskSpace.textContent = `${adapted.disk_used.toFixed(1)} GB / ${adapted.disk_total.toFixed(1)} GB`;

            // Calc steps
            state.steps.cpu = (state.data.cpu - state.current.cpu) / ANIMATION_FRAMES;
            state.steps.temp = (state.data.temp - state.current.temp) / ANIMATION_FRAMES;
            state.steps.mem = (state.data.mem - state.current.mem) / ANIMATION_FRAMES;
            state.steps.disk = (state.data.disk - state.current.disk) / ANIMATION_FRAMES;

            // Trigger UI
            UIManager.updateUI();
            ChartManager.updateCharts();
        }
    };

    // UI Manager
    const UIManager = {
        setConnectionStatus(connected) {
            elements.connectionText.innerHTML = connected ? "已连接" : "连接断开";
            elements.connectionStatus.classList.toggle("connected", connected);
            elements.connectionStatus.classList.toggle("pulse", connected);
        },

        updateUI() {
            this.animateGauges();
            this.updateStatus();
            this.updateTempDesc();
        },

        animateGauges() {
            const animate = () => {
                let complete = true;
                const keys = ['cpu', 'temp', 'mem', 'disk'];

                keys.forEach(key => {
                    const target = state.data[key];
                    const diff = target - state.current[key];
                    if (Math.abs(diff) > 0.5) {
                        state.current[key] += state.steps[key];
                        complete = false;
                    } else {
                        state.current[key] = target;
                    }
                    this.renderGauge(key, state.current[key]);
                });

                if (!complete) requestAnimationFrame(animate);
            };

            requestAnimationFrame(animate);
        },

        renderGauge(key, value) {
            const config = UNITS[key];
            const clamped = Math.max(0, Math.min(value, config.max));
            const deg = clamped * config.factor;
            const percentage = Math.round(clamped);

            // Fixed: add comma in conic-gradient syntax
            elements[key].style.background = `conic-gradient(var(--primary) 0deg, var(--primary) ${deg}deg, var(--card-bg) ${deg}deg, var(--card-bg) 360deg)`;

            // Fixed: remove -90 offset to match original alignment
            elements[`${key}Angle`].style.transform = `rotate(${deg}deg)`;

            elements[`${key}Params`].innerHTML = `${percentage}${config.suffix}`;
        },

        updateStatus() {
            const keys = ['cpu', 'temp', 'mem', 'disk'];
            keys.forEach(key => {
                const statusEl = elements[`${key}Status`];
                if (!statusEl) return;
                const value = state.data[key];
                const { text, class: cls } = DataAdapter.getStatusConfig(key, value);
                statusEl.textContent = text;
                statusEl.className = `metric-status ${cls}`;
            });
        },

        updateTempDesc() {
            if (!elements.tempDesc) return;
            const temp = state.data.temp;
            if (temp > 70) {
                elements.tempDesc.textContent = "温度过高!"; elements.tempDesc.style.color = "var(--danger)";
            } else if (temp > 60) {
                elements.tempDesc.textContent = "温度升高"; elements.tempDesc.style.color = "var(--warning)";
            } else {
                elements.tempDesc.textContent = "温度正常"; elements.tempDesc.style.color = "var(--text-secondary)";
            }
        }
    };

    // Chart Manager
    const ChartManager = {
        updateCharts() {
            this.updateChart(elements.cpuChart, state.chartData.cpu);
            this.updateChart(elements.tempChart, state.chartData.temp);
            this.updateChart(elements.memChart, state.chartData.mem);
            this.updateChart(elements.diskChart, state.chartData.disk);
        },

        updateChart(chartElement, data) {
            if (!chartElement) return;
            chartElement.innerHTML = '';
            const MAX_VALUE = 100;
            for (let i = 0; i < data.length; i++) {
                const bar = document.createElement('div');
                bar.className = 'chart-bar';
                const height = Math.min(data[i], MAX_VALUE);
                bar.style.height = `${height}%`;
                bar.style.background = data[i] > 90 ? "var(--danger)" : data[i] > 80 ? "var(--warning)" : "var(--primary)";
                chartElement.appendChild(bar);
            }
        }
    };

    // System Info Manager
    const SystemInfoManager = {
        fetchSystemInfo() {
            fetch(INFO_ENDPOINT)
                .then(response => {
                    if (!response.ok) {
                        throw new Error(`HTTP error! status: ${response.status}`);
                    }
                    return response.json();
                })
                .then(data => {
                    // Update DOM elements
                    if (infoElements.sysName) infoElements.sysName.textContent = `${data.os} ${data.platform}`;
                    if (infoElements.sysKernel) infoElements.sysKernel.textContent = data.kernel;
                    if (infoElements.sysCpu) infoElements.sysCpu.textContent = data.cpu_model;
                    if (infoElements.sysSoc) infoElements.sysSoc.textContent = data.cpu_specs;
                    if (infoElements.sysMem) infoElements.sysMem.textContent = `${Math.round(data.mem_total_gb)} GB`;
                    if (infoElements.sysDisk) infoElements.sysDisk.textContent = `${Math.round(data.disk_total_gb)} GB`;

                    // Set boot time for uptime calculation
                    systemInfo.startTime = Date.now() - (data.uptime_seconds * 1000);

                    // Start uptime counter
                    this.startUptimeCounter();
                })
                .catch(err => {
                    console.error('Failed to fetch system info:', err);
                    // Fallback to N/A
                    Object.values(infoElements).forEach(el => el.textContent = "N/A");
                });
        },

        updateUptimeDisplay() {
            const totalSeconds = Math.floor((Date.now() - systemInfo.startTime) / 1000);
            const days = Math.floor(totalSeconds / (3600 * 24));
            const hours = Math.floor((totalSeconds % (3600 * 24)) / 3600);
            const minutes = Math.floor((totalSeconds % 3600) / 60);
            if (infoElements.sysUptime) {
                infoElements.sysUptime.textContent = `${days}天 ${hours}小时 ${minutes}分`;
            }
        },

        startUptimeCounter() {
            if (systemInfo.uptimeInterval) clearInterval(systemInfo.uptimeInterval);
            systemInfo.uptimeInterval = setInterval(() => this.updateUptimeDisplay(), 1000);
            this.updateUptimeDisplay();
        }
    };

    // Refresh Manager
    const RefreshManager = {
        setup() {
            if (!elements.refreshBtn) return;
            elements.refreshBtn.addEventListener('click', () => {
                elements.refreshBtn.style.transition = 'transform 0.5s ease';
                elements.refreshBtn.style.transform = 'rotate(360deg)';
                setTimeout(() => elements.refreshBtn.style.transform = 'rotate(0deg)', 500);
                
                const rawMock = {
                    cpu_usage: Math.random() * 100,
                    temperature: Math.random() * 70 + 30,
                    memory_usage: Math.random() * 100,
                    storage_usage: Math.random() * 100,
                    cpu_frequency: Math.floor(Math.random() * 500) + 800,
                    memory_used: Math.floor(Math.random() * 800) + 200,
                    memory_total: 1000,
                    storage_used: Math.floor(Math.random() * 25) + 7,
                    storage_total: 32
                };
                const adapted = DataAdapter.adapt(rawMock);
                StateManager.updateState(adapted);
            });
        }
    };

    // Init
    function init() {
        cacheElements();
        SystemInfoManager.fetchSystemInfo();
        const connectionManager = new ConnectionManager();
        connectionManager.connect();
        ChartManager.updateCharts();
        RefreshManager.setup();
        window.connectionManager = connectionManager; // Debug
    }

    return { init };
})();

window.addEventListener('DOMContentLoaded', MonitorSystem.init);
window.addEventListener('beforeunload', () => window.connectionManager?.disconnect());