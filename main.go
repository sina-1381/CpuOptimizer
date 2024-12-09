package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	minimumTemperture = 40
	maximumTemperture = 75
)

var (
	cpuMinimumFrequency = executeCommand("lscpu | awk '/min/ {print $NF}'", true)
	cpuMaximumFrequency = executeCommand("lscpu | awk '/max/ {print $NF}'", true)
	gpuMinimumFrequency = executeCommand("cat /sys/class/drm/card1/gt_RP1_freq_mhz", true)
	gpuMaximumFrequency = executeCommand("cat /sys/class/drm/card1/gt_RP0_freq_mhz", true)
)

func main() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			go func() {
				applyFrequencies(getCurrentTemperture())
			}()
		}
	}
}

func executeCommand(command string, out bool) int {
	response, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		log.Println(err)
	}
	if out == true {
		str := strings.TrimSpace(string(response))
		output, err := strconv.ParseFloat(str, 64)
		if err != nil {
			log.Println(err)
		}
		return int(output)
	} else {
		return 0
	}
}

func applyFrequencies(currentTemperture int) {
	_ = executeCommand(cpuFrequencyCommand(calculateSafeCpuFrequency(currentTemperture)), false)
	_ = executeCommand(gpuFrequencyCommand(calculateSafeGpuFrequency(currentTemperture)), false)
}

func getCurrentTemperture() int {
	return executeCommand("cat /sys/class/thermal/thermal_zone0/temp", true) / 1000
}

func calculateSafeCpuFrequency(currentTemperture int) int {
	if currentTemperture >= maximumTemperture {
		return cpuMinimumFrequency
	}

	if currentTemperture <= minimumTemperture {
		return cpuMaximumFrequency
	}

	return (cpuMaximumFrequency - ((cpuMaximumFrequency-cpuMinimumFrequency)/(maximumTemperture-minimumTemperture))*(currentTemperture-minimumTemperture)) * 1000
}

func calculateSafeGpuFrequency(currentTemperture int) int {
	if currentTemperture >= maximumTemperture {
		return gpuMinimumFrequency
	}

	if currentTemperture <= minimumTemperture {
		return gpuMaximumFrequency
	}

	return (gpuMaximumFrequency - ((gpuMaximumFrequency-(gpuMinimumFrequency))/(maximumTemperture-minimumTemperture))*(currentTemperture-minimumTemperture))
}

func cpuFrequencyCommand(frequency int) string {
	return fmt.Sprintf(
		`for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_max_freq; do
echo %d | tee $cpu;
done`, frequency)
}

func gpuFrequencyCommand(frequency int) string {
	return fmt.Sprintf(`echo %d | tee /sys/class/drm/card1/gt_max_freq_mhz`, frequency)
}

// sudo chmod 777 /sys/devices/system/cpu/cpu*/cpufreq/scaling_max_freq /sys/class/drm/card1/gt_max_freq_mhz

// sudo nano /etc/systemd/system/cpuoptimizer.service

// sudo systemctl daemon-reload

// sudo systemctl enable --now cpuoptimizer.service

/*
[Unit]
Description=optimizing cpu/gpu frequency
After=network.target

[Service]
ExecStartPre=/bin/bash -c ''
ExecStart=/home/sina/go/src/CpuOptimizer/CpuOptimizer
Restart=always

[Install]
WantedBy=multi-user.target
*/
