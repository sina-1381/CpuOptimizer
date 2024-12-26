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
	err         error
	preferences map[string]map[string]any
	gpuMinFreq  int
	gpuMaxFreq  int
	systemMode  string
)

func main() {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			currentTemp := CurrentTemp()
			setSettingsBasedOnTemp(currentTemp)
		}
	}
}

func init() {
	gpuMinFreq, gpuMaxFreq = generateGpuOutput(executeCommand("cat /sys/class/drm/card*/gt_RP1_freq_mhz /sys/class/drm/card*/gt_RP0_freq_mhz"))

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
			"min_temp":   60,
			"max_temp":   70,
		},
		"performance": {
			"cpu_status": "balance_performance",
			"gpu_freq":   gpuFrequenCal(3),
			"min_temp":   0,
			"max_temp":   59,
		},
	}
}

func CurrentTemp() int {
	return generateTempOutput(executeCommand("cat /sys/class/thermal/thermal_zone*/temp"))
}

func setSettingsBasedOnTemp(currentTemp int) {
	for mode, settings := range preferences {
		if currentTemp >= settings["min_temp"].(int) && currentTemp <= settings["max_temp"].(int) {
			if systemMode != mode {
				executeCommand(applySettingsCommand(settings["cpu_status"].(string), settings["gpu_freq"].(int)))
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

func generateTempOutput(response []byte) int {
	temps := strings.Fields(string(response))
	var sum int
	for _, temp := range temps {
		num, err := strconv.Atoi(temp)
		if err != nil {
			log.Printf("system error : '%s'", err)
		}
		sum += num
	}
	return (sum / len(temps)) / 1000
}

func generateGpuOutput(response []byte) (int, int) {
	info := strings.Fields(string(response))
	var out []int
	for _, value := range info {
		num, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("system error : '%s'", err)
		}
		out = append(out, num)
	}
	return out[0], out[1]
}

func applySettingsCommand(preference string, frequency int) string {
	return fmt.Sprintf(`echo %s | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/energy_performance_preference && echo %d | tee /sys/class/drm/card*/gt_max_freq_mhz`, preference, frequency)
}

func gpuFrequenCal(multi int) int {
	return gpuMinFreq + (((gpuMaxFreq - gpuMinFreq) / 3) * multi)
}

/*
sudo nano /etc/systemd/system/cpuoptimizer.service
sudo systemctl daemon-reload
sudo systemctl enable --now cpuoptimizer.service

[Unit]
Description=optimizing cpu/gpu frequency
After=network.target

[Service]
ExecStartPre=/bin/bash -c 'sudo chmod 777 /sys/devices/system/cpu/cpu./cpufreq/energy_performance_preference /sys/class/drm/card./gt_max_freq_mhz'
ExecStart=/home/sina/go/src/CpuOptimizer/CpuOptimizer
Restart=on-failure

[Install]
WantedBy=multi-user.target
*/
