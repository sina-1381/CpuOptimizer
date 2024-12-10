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
	defaultSafeTemperature     = 55
	minimumTemperature         = 35
	maximumTemperature         = 75
	temperatureChangeThreshold = 2
	tickerTimeDefaultValue     = 5
	maximumTickerTime          = 30
)

const (
	getCpuMinimumFrequencyCommand = "lscpu | awk '/min/ {print $NF}'"
	getCpuMaximumFrequencyCommand = "lscpu | awk '/max/ {print $NF}'"
	getGpuMinimumFrequencyCommand = "cat /sys/class/drm/card1/gt_RP1_freq_mhz"
	getGpuMaximumFrequencyCommand = "cat /sys/class/drm/card1/gt_RP0_freq_mhz"
	getSystemTemperatureCommand   = "cat /sys/class/thermal/thermal_zone0/temp"
)

var err error

var (
	cpuMinimumFrequency int
	cpuMaximumFrequency int
	gpuMinimumFrequency int
	gpuMaximumFrequency int
	previousTemperature int
	tickerTime          int
)

func main() {
	ticker := time.NewTicker(time.Duration(tickerTime) * time.Second)
	for {
		select {
		case <-ticker.C:
			currentTemperature := getCurrentTemperature()

			if abs(currentTemperature-previousTemperature) >= temperatureChangeThreshold {
				applyFrequencies(currentTemperature)
				previousTemperature = currentTemperature
				tickerTime = tickerTimeDefaultValue
			} else {
				tickerTime++
				if tickerTime >= maximumTickerTime {
					tickerTime = tickerTimeDefaultValue
				}
			}
		}
	}
}

func init() {
	cpuMinimumFrequency, err = executeCommand(getCpuMinimumFrequencyCommand, true)
	if err != nil {
		log.Printf("couldn't get system's minimum CPU frequency: %v", err)
	}
	cpuMaximumFrequency, err = executeCommand(getCpuMaximumFrequencyCommand, true)
	if err != nil {
		log.Printf("couldnt get systems maximum cpu frequency: %v", err)
	}
	gpuMinimumFrequency, err = executeCommand(getGpuMinimumFrequencyCommand, true)
	if err != nil {
		log.Printf("couldnt get systems minimum gpu frequency: %v", err)
	}
	gpuMaximumFrequency, err = executeCommand(getGpuMaximumFrequencyCommand, true)
	if err != nil {
		log.Printf("couldnt get systems maximum gpu frequency: %v", err)
	}
	tickerTime = tickerTimeDefaultValue
}

func executeCommand(command string, out bool) (int, error) {
	response, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("couldn't execute command '%s': %v", command, err)
	}

	if out == true {
		str := strings.TrimSpace(string(response))
		output, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return 0, fmt.Errorf("couldn't parse command output for '%s': %v", command, err)
		}
		return int(output), nil
	}

	return 0, nil
}

func applyFrequencies(currentTemperature int) {
	_, err = executeCommand(cpuFrequencyCommand(calculateSafeCpuFrequency(currentTemperature)), false)
	if err != nil {
		log.Printf("couldn't apply CPU frequency: %v", err)
	}
	_, err = executeCommand(gpuFrequencyCommand(calculateSafeGpuFrequency(currentTemperature)), false)
	if err != nil {
		log.Printf("couldn't apply GPU frequency: %v", err)
	}
}

func getCurrentTemperature() int {
	temperature, err := executeCommand(getSystemTemperatureCommand, true)
	if err != nil {
		log.Printf("couldn't read system temperature, using default value: %v", err)
		return defaultSafeTemperature
	}
	return temperature / 1000 // converting to celsius
}

func calculateSafeCpuFrequency(currentTemperature int) int {
	if currentTemperature >= maximumTemperature {
		return cpuMinimumFrequency
	}

	if currentTemperature <= minimumTemperature {
		return cpuMaximumFrequency
	}

	return (cpuMaximumFrequency - ((cpuMaximumFrequency-cpuMinimumFrequency)/(maximumTemperature-minimumTemperature))*(currentTemperature-minimumTemperature)) * 1000
}

func calculateSafeGpuFrequency(currentTemperature int) int {
	if currentTemperature >= maximumTemperature {
		return gpuMinimumFrequency
	}

	if currentTemperature <= minimumTemperature {
		return gpuMaximumFrequency
	}

	return (gpuMaximumFrequency - ((gpuMaximumFrequency-(gpuMinimumFrequency))/(maximumTemperature-minimumTemperature))*(currentTemperature-minimumTemperature))
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

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
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
