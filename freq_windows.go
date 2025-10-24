//go:build windows

package main

import (
	"errors"

	"github.com/shirou/gopsutil/v4/cpu"
)

func GetCPUFreq() (int, error) {
	infos, err := cpu.Info()
	if err != nil {
		return 0, err
	}
	if len(infos) == 0 {
		return 0, errors.New("无法获取CPU信息")
	}
	// 返回第一个作为主CPU的频率
	return int(infos[0].Mhz), nil
}
