package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type fanHardware struct {
	temperatures map[string]string
	channels     map[string]linuxFan
}

type linuxFan struct{ pwm, rpm string }

var (
	pwmName  = regexp.MustCompile("^pwm([0-9]+)$")
	tempName = regexp.MustCompile("^temp[0-9]+_input$")
)

func elevated() bool                { return os.Geteuid() == 0 }
func (*FanService) RestartAsAdmin() {}
func (*FanService) closeHardware()  {}

func readHwmon(path string) string {
	data, _ := os.ReadFile(path)
	return strings.TrimSpace(string(data))
}

func hwmonNumber(path string) int {
	value, err := strconv.Atoi(readHwmon(path))
	if err != nil {
		return -1
	}
	return value
}

func writeHwmon(path string, value int) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(file, value)
	return errors.Join(err, file.Close())
}

// The watchdog inherits an already-open sysfs descriptor. If this app is
// killed, closing its pipe triggers a best-effort return to automatic mode.
func startFanWatchdog(mode string, original int) (*os.File, error) {
	modeFile, err := os.OpenFile(mode, os.O_WRONLY, 0)
	if err != nil {
		return nil, err
	}
	defer modeFile.Close()
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	read, write, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer read.Close()
	cmd := exec.Command(exe, "--linux-fan-restore", strconv.Itoa(original))
	cmd.Stdin, cmd.Stderr = read, os.Stderr
	cmd.ExtraFiles = []*os.File{modeFile}
	if err := cmd.Start(); err != nil {
		write.Close()
		return nil, err
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("fan restore watchdog: %v", err)
		}
	}()
	return write, nil
}

func restoreFanOnExit(original int) error {
	mode := os.NewFile(3, "fan PWM mode")
	defer mode.Close()
	var command [1]byte
	if _, err := io.ReadFull(os.Stdin, command[:]); err == nil && command[0] == 'q' {
		return nil
	}
	_, err := fmt.Fprintln(mode, original)
	return err
}

func (f *FanService) detect() {
	f.hardware = fanHardware{temperatures: map[string]string{}, channels: map[string]linuxFan{}}
	dirs, _ := filepath.Glob("/sys/class/hwmon/hwmon*")
	for _, dir := range dirs {
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			continue
		}
		chip := readHwmon(filepath.Join(dir, "name"))
		if chip == "" {
			continue
		}
		stable := strings.TrimPrefix(filepath.Dir(filepath.Dir(real)), "/sys/devices/") + "/" + chip
		entries, _ := os.ReadDir(dir)
		for _, entry := range entries {
			name := entry.Name()
			if !tempName.MatchString(name) {
				continue
			}
			path := filepath.Join(real, name)
			temp := hwmonNumber(path)
			if temp < 0 || temp > 150000 {
				continue
			}
			label := readHwmon(filepath.Join(real, strings.TrimSuffix(name, "_input")+"_label"))
			if label == "" {
				label = chip + " " + strings.TrimSuffix(name, "_input")
			}
			if strings.HasPrefix(label, "Package id") || label == "Tctl" {
				label = "CPU"
			}
			id := "hwmon:" + stable + "/" + name
			f.hardware.temperatures[id] = path
			f.sensors = append(f.sensors, TempSensor{ID: id, Name: label, Temp: temp / 1000})
		}
		for _, entry := range entries {
			name := entry.Name()
			match := pwmName.FindStringSubmatch(name)
			if match == nil {
				continue
			}
			pwm := filepath.Join(real, name)
			mode := pwm + "_enable"
			original := hwmonNumber(mode)
			// Only take over channels already using automatic control, so Stop has
			// a known firmware mode to restore. Both attributes must be writable.
			if original < 2 || unix.Access(mode, unix.W_OK) != nil || unix.Access(pwm, unix.W_OK) != nil {
				continue
			}
			id := "hwmon:" + stable + "/" + name
			rpm := filepath.Join(real, "fan"+match[1]+"_input")
			label := readHwmon(filepath.Join(real, "fan"+match[1]+"_label"))
			if label == "" {
				label = "Fan #" + match[1]
			}
			f.hardware.channels[id] = linuxFan{pwm: pwm, rpm: rpm}
			preferredSensor := ""
			if chip == "amdgpu" {
				preferredSensor = "hwmon:" + stable + "/temp1_input"
			}
			var watchdog *os.File
			restore := func() error {
				if watchdog == nil {
					return nil
				}
				if err := writeHwmon(mode, original); err != nil {
					return err
				}
				_, _ = watchdog.Write([]byte{'q'})
				_ = watchdog.Close()
				watchdog = nil
				return nil
			}
			f.fans = append(f.fans, &fanDev{
				Fan:             Fan{ID: id, Name: label, Source: chip, MinDuty: mbMinDuty, RPM: -1, Duty: -1},
				preferredSensor: preferredSensor,
				applied:         -1,
				set: func(percent int) error {
					if watchdog == nil {
						if hwmonNumber(mode) != original {
							return errors.New("PWM control mode changed since startup")
						}
						guard, err := startFanWatchdog(mode, original)
						if err != nil {
							return err
						}
						watchdog = guard
						if err := writeHwmon(mode, 1); err != nil {
							return errors.Join(err, restore())
						}
					}
					if err := writeHwmon(pwm, (percent*255+50)/100); err != nil {
						return errors.Join(err, restore())
					}
					return nil
				},
				restore: restore,
			})
		}
	}
	if len(f.fans) == 0 {
		f.notice = "No writable automatic PWM channels found. Linux fan control needs a compatible hwmon driver and permission to write its PWM controls."
	}
}

func (f *FanService) readHardware() map[string]int {
	temps := make(map[string]int, len(f.hardware.temperatures))
	for id, path := range f.hardware.temperatures {
		temp := hwmonNumber(path)
		if temp >= 0 && temp <= 150000 {
			temps[id] = temp / 1000
		} else {
			temps[id] = -1
		}
	}
	for _, d := range f.fans {
		channel := f.hardware.channels[d.ID]
		d.RPM, d.Duty = hwmonNumber(channel.rpm), -1
		if pwm := hwmonNumber(channel.pwm); pwm >= 0 && pwm <= 255 {
			d.Duty = (100*pwm + 127) / 255
		}
	}
	return temps
}
