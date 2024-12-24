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
	getSystemTemperatureCommand   = "cat /sys/class/thermal/thermal_zone*/temp"
	getGpuMinimumFrequencyCommand = "cat /sys/class/drm/card*/gt_RP1_freq_mhz"
	getGpuMaximumFrequencyCommand = "cat /sys/class/drm/card*/gt_RP0_freq_mhz"
)

var (
	err                 error
	preferences         map[string]map[string]any
	gpuMinimumFrequency int
	gpuMaximumFrequency int
)

func main() {
	preferences = map[string]map[string]any{
		"power": {
			"cpu_status": "power",
			"gpu_freq":   gpuFrequenCal(1),
			"min_temp":   71,
			"max_temp":   200,
		},
		"balance": {
			"cpu_status": "balance_power",
			"gpu_freq":   gpuFrequenCal(2),
			"min_temp":   56,
			"max_temp":   70,
		},
		"performance": {
			"cpu_status": "balance_performance",
			"gpu_freq":   gpuFrequenCal(3),
			"min_temp":   0,
			"max_temp":   55,
		},
	}

	ticker := time.NewTicker(time.Minute * 1)
	for {
		select {
		case <-ticker.C:
			currentTemp := getCurrentTemp()
			setSettingsBasedOnTemp(currentTemp)
		}
	}
}

func init() {
	gpuMinimumFrequency, err = executeCommand(getGpuMinimumFrequencyCommand, "info")
	if err != nil {
		log.Printf("couldnt get systems minimum gpu frequency: %v", err)
	}
	gpuMaximumFrequency, err = executeCommand(getGpuMaximumFrequencyCommand, "info")
	if err != nil {
		log.Printf("couldnt get systems maximum gpu frequency: %v", err)
	}
}

func getCurrentTemp() int {
	temperature, err := executeCommand(getSystemTemperatureCommand, "temp")
	if err != nil {
		log.Printf("couldn't read system temperature, using default value: %v", err)
	}
	return temperature
}

func cpuPreferenceCommand(preference string) string {
	return fmt.Sprintf(`echo %s | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/energy_performance_preference`, preference)
}

func gpuFrequencyCommand(frequency int) string {
	return fmt.Sprintf(`echo %d | tee /sys/class/drm/card*/gt_max_freq_mhz`, frequency)
}

func gpuFrequenCal(multi int) int {
	a := (gpuMaximumFrequency - gpuMinimumFrequency) / 3
	return gpuMinimumFrequency + (a * multi)
}

func setSettingsBasedOnTemp(currentTemp int) {
	var cpuStatus string
	var gpuFreq int

	for _, settings := range preferences {
		if currentTemp >= settings["min_temp"].(int) && currentTemp <= settings["max_temp"].(int) {
			cpuStatus = settings["cpu_status"].(string)
			gpuFreq = settings["gpu_freq"].(int)
		}
	}
	_, err = executeCommand(cpuPreferenceCommand(cpuStatus), "")
	if err != nil {
		log.Printf("Failed to set CPU status to '%s': %v", cpuStatus, err)
	}
	_, err = executeCommand(gpuFrequencyCommand(gpuFreq), "")
	if err != nil {
		log.Printf("Failed to set GPU frequency to '%d': %v", gpuFreq, err)
	}
}

func executeCommand(command string, mode string) (int, error) {
	response, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("couldn't execute command '%s': %v", command, err)
	}

	switch mode {
	case "temp":
		temps := strings.Fields(string(response))
		var sum int
		for _, temp := range temps {
			num, err := strconv.Atoi(temp)
			if err != nil {
				return 0, fmt.Errorf("system error : '%s'", err)
			}
			sum += num
		}
		avarage := (sum / len(temps)) / 1000
		return avarage, nil

	case "info":
		output, err := strconv.ParseFloat(strings.TrimSpace(string(response)), 64)
		if err != nil {
			return 0, fmt.Errorf("couldn't parse command output for '%s': %v", command, err)
		}
		return int(output), nil

	default:
		return 0, nil
	}
}

// sudo chmod 777 /sys/devices/system/cpu/cpu*/cpufreq/energy_performance_preference /sys/class/drm/card*/gt_max_freq_mhz

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
