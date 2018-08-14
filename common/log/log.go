package log

import "fmt"

func Info(format string) {
	fmt.Printf("Info: %s.\n", format)
}

func Warn(format string) {
	fmt.Printf("Warn: %s.\n", format)
}

func Error(format string) {
	fmt.Printf("Error: %s.\n", format)
}