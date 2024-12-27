package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	err          error
	systemMode   string
	cpuCores     []map[string]int
	cpuCoreCount int
	modes        map[string]map[string]any
	gpuMinFreq   int
	gpuMaxFreq   int
	cpuMinFreq   int
	cpuMaxFreq   int
)

func main() {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			currentTemp := CurrentTemp()
			setSettingsBasedOnTemp(currentTemp)
		}
	}
}

func init() {
	gpuMinFreq, gpuMaxFreq = getGpuInfo(generateIntegerOutput(executeCommand("cat /sys/class/drm/card*/gt_RP1_freq_mhz /sys/class/drm/card*/gt_RP0_freq_mhz")))
	cpuCoreCount = getCpuCoreCount(generateIntegerOutput(executeCommand("cat /proc/cpuinfo | grep ^processor | wc -l")))
	cpuCores = getCpuCoresInfo(generateIntegerOutput(executeCommand("cat /sys/devices/system/cpu/cpu*/cpufreq/cpuinfo_min_freq /sys/devices/system/cpu/cpu*/cpufreq/cpuinfo_max_freq")))
	cpuMinFreq, cpuMaxFreq = cpuCores[0]["min_freq"], cpuCores[0]["max_freq"]
	modes = modesData()
}

func CurrentTemp() int {
	return getTemp(generateIntegerOutput(executeCommand("cat /sys/class/thermal/thermal_zone*/temp")))
}

func setSettingsBasedOnTemp(currentTemp int) {
	for mode, settings := range modes {
		if currentTemp >= settings["min_temp"].(int) && currentTemp <= settings["max_temp"].(int) {
			if systemMode != mode {
				executeCommand(applySettingsCommand(settings["cpu_status"].(string), settings["gpu_freq"].(int), settings["cpu_freq"].(int)))
				systemMode = mode
				return
			} else {
				return
			}
		}
	}
}

func executeCommand(command string) []byte {
	response, err := exec.Command("sh", "-c", command).CombinedOutput()
	if err != nil {
		log.Printf("couldn't execute command '%s': %v", command, err)
	}
	return response
}

func generateIntegerOutput(response []byte) []int {
	data := strings.Fields(string(response))
	var out []int
	for _, value := range data {
		num, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("system error : '%s'", err)
		}
		out = append(out, num)
	}
	return out
}

func getGpuInfo(input []int) (int, int) {
	min_freq := input[0]
	max_freq := input[1]

	return min_freq, max_freq
}

func getCpuCoreCount(input []int) int {
	coreCount := input[0]

	return coreCount
}

func getCpuCoresInfo(input []int) []map[string]int {
	var out []map[string]int
	for i := 0; i < cpuCoreCount; i++ {
		out = append(out, map[string]int{
			"min_freq": input[i],
			"max_freq": input[i+cpuCoreCount],
		})
	}
	return out
}

func getTemp(input []int) int {
	var temp int
	for _, value := range input {
		temp += value
	}
	return (temp / len(input)) / 1000
}

func applySettingsCommand(preference string, gpuFreq, cpuFreq int) string {
	return fmt.Sprintf(`echo %s | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/energy_performance_preference && echo %d | tee /sys/class/drm/card*/gt_max_freq_mhz && echo %d | tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_max_freq`, preference, gpuFreq, cpuFreq)
}

func balanceRate(max, min int) int {
	return (max - min) / 5
}

func average(max, min int) int {
	return (max + min) / 2
}

func modesData() map[string]map[string]any {
	out := map[string]map[string]any{
		"power": {
			"cpu_status": "power",
			"cpu_freq":   cpuMinFreq + balanceRate(cpuMaxFreq, cpuMinFreq),
			"gpu_freq":   gpuMinFreq + balanceRate(gpuMaxFreq, gpuMinFreq),
			"min_temp":   71,
			"max_temp":   200,
		},
		"balance": {
			"cpu_status": "balance_power",
			"cpu_freq":   average(cpuMaxFreq, cpuMinFreq),
			"gpu_freq":   average(gpuMaxFreq, gpuMinFreq),
			"min_temp":   60,
			"max_temp":   70,
		},
		"performance": {
			"cpu_status": "balance_performance",
			"cpu_freq":   cpuMaxFreq - balanceRate(cpuMaxFreq, cpuMinFreq),
			"gpu_freq":   gpuMaxFreq - balanceRate(gpuMaxFreq, gpuMinFreq),
			"min_temp":   0,
			"max_temp":   59,
		},
	}
	return out
}

/*
sudo nano /etc/systemd/system/cpuoptimizer.service
sudo systemctl daemon-reload
sudo systemctl enable --now cpuoptimizer.service

[Unit]
Description=optimizing cpu/gpu frequency
After=network.target

[Service]
ExecStartPre=/bin/bash -c 'sudo chmod 777 /sys/devices/system/cpu/cpu./cpufreq/energy_performance_preference /sys/class/drm/card./gt_max_freq_mhz /sys/devices/system/cpu/cpu./cpufreq/scaling_max_freq'
ExecStart=/home/sina/go/src/CpuOptimizer/CpuOptimizer
Restart=on-failure

[Install]
WantedBy=multi-user.target
*/
