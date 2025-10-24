//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

func GetCPUFreq() (int, error) {
	path := "/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq"
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("读取Linux CPU频率文件失败: %w", err)
	}
	if len(data) == 0 {
		return 0, errors.New("Linux CPU频率文件为空")
	}
	rawdata, _ := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("解析Linux CPU频率失败: %w", err)
	}
	return rawdata / 1000, nil
}
