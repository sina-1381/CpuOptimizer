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
	minimumTemperture = 40
	maximumTemperture = 70
	minimumFrequency  = executeCommand("lscpu | awk '/min/ {print $NF}'", true)
	maximumFrequency  = executeCommand("lscpu | awk '/max/ {print $NF}'", true)
)

func main() {
	//sudo chmod 777 -R /sys/devices/system/cpu/cpu*/cpufreq/scaling_max_freq
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			go func() {
				currentTemperture := executeCommand("cat /sys/class/thermal/thermal_zone0/temp", true) / 1000
				newFrequency := calculateSafeFrequency(currentTemperture)
				changeFrequencyCommand := fmt.Sprintf(
					`for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_max_freq; do
  echo %d | tee $cpu;
done`, newFrequency)
				_ = executeCommand(changeFrequencyCommand, false)
				fmt.Println(newFrequency, currentTemperture)
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

func calculateSafeFrequency(currentTemperture int) int {
	return (maximumFrequency - ((maximumFrequency-minimumFrequency)/(maximumTemperture-minimumTemperture))*(currentTemperture-minimumTemperture)) * 1000
}
