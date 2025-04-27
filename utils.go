package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

var magic_space string = "                                                            "
var magic_backspace string = "\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b"

func colorString(inputString string, textcolor string, backgroundcolor string) string {
	textcolorMap := map[string]string{
		"None":         "30",
		"Red":          "31",
		"Green":        "32",
		"Yellow":       "33",
		"Blue":         "34",
		"Magenta":      "35",
		"Cyan":         "36",
		"Light gray":   "37",
		"Dark gray":    "90",
		"Light red":    "91",
		"Light green":  "92",
		"Light yellow": "93",
		"Light blue":   "94",
		"Light agenta": "95",
		"Light cyan":   "96",
		"White":        "97",
		"Bold":         "1",
		"Underline":    "4",
		"No underline": "24",
		"Negative":     "7",
		"Positive":     "27",
		"Default":      "0",
	}

	backgroundcolorMap := map[string]string{
		"Black":        ";40",
		"Red":          ";41",
		"Green":        ";42",
		"Yellow":       ";43",
		"Blue":         ";44",
		"Magenta":      ";45",
		"Cyan":         ";46",
		"Light gray":   ";47",
		"Dark gray":    ";100",
		"Light red":    ";101",
		"Light green":  ";102",
		"Light yellow": ";103",
		"Light blue":   ";104",
		"Light agenta": ";105",
		"Light cyan":   ";106",
		"White":        ";107",
		"None":         "",
	}

	return magic_backspace + "[" + textcolorMap[textcolor] + backgroundcolorMap[backgroundcolor] + "m" + inputString + "[40;0m" + strings.Replace(magic_space, " ", "", len(inputString)+len(textcolorMap[textcolor])+len(backgroundcolorMap[backgroundcolor])-3)
}

func currentDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Unable to get the current filename")
	}
	return filepath.Dir(filename)
}

func execWrapper(commandLine string) {
	comSpec := os.Getenv("COMSPEC")
	if comSpec == "" {
		comSpec = os.Getenv("SystemRoot") + "\\System32\\cmd.exe"
	}

	childProcess := exec.Command(comSpec)
	childProcess.SysProcAttr = &syscall.SysProcAttr{CmdLine: "/C \"" + commandLine + "\""}

	childProcess.Run()
}

func sortVideoByVal(inputMap map[int]string) []int {
	qualities := []string{"hd1080", "hd720", "large", "medium", "small", "tiny"}

	inVideokeys := make([]int, 0, len(inputMap))
	for key := range inputMap {
		inVideokeys = append(inVideokeys, key)
	}

	outVideokeys := make([]int, 0, len(inputMap))
	for _, i := range inVideokeys {
		for _, q := range qualities {
			if strings.Contains(inputMap[i], q) {
				outVideokeys = append(outVideokeys, i)
			}
		}
	}
	sort.Ints(outVideokeys)
	return outVideokeys
}

func sortAudioByVal(inputMap map[int]string) int {
	Audiokeys := make([]int, 0, len(inputMap))
	for key := range inputMap {
		Audiokeys = append(Audiokeys, key)
	}
	return Audiokeys[0]
}

func removeInvalidFilenameChars(filename string) string {
	invalidChars := "\\/:*?\"<>|"
	for _, v := range invalidChars {
		filename = strings.ReplaceAll(filename, string(v), "")
	}
	return filename
}

func checkForPlaylist(slice []string) string {
	for _, v := range slice {
		if strings.Contains(v, "list") {
			return v
		}
	}
	return "false"
}

func strToInt(str string) int {
	outInt, err := strconv.Atoi(str)
	if err != nil {
		panic(err)
	}
	return outInt
}

func readFile(filepath string) string {
	data, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func writeFile(filepath string, data string) {
	err := os.WriteFile(filepath, []byte(data), 0644)
	if err != nil {
		panic(err)
	}
}

func appendFile(filepath string, data string) {
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := f.WriteString(data); err != nil {
		panic(err)
	}
}

func clamp(num int, nummin int, nummax int) int {
	return min(max(num, nummax), nummin)
}

func min(num int, min int) int {
	if num < min {
		return min
	}
	return num
}

func max(num int, max int) int {
	if num > max {
		return max
	}
	return num
}
