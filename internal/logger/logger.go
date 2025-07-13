package logger

import (
	"fmt"
	"log"
	"os"
)

var (
	debugLogger *log.Logger
	infoLogger  *log.Logger
	errorLogger *log.Logger
	verbose     bool
)

func init() {
	debugLogger = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLogger = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime)
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// SetVerbose は詳細出力モードを設定する
func SetVerbose(v bool) {
	verbose = v
}

// Debug はデバッグメッセージを出力する（verboseモードのみ）
func Debug(format string, args ...interface{}) {
	if verbose {
		debugLogger.Output(2, fmt.Sprintf(format, args...))
	}
}

// Info は情報メッセージを出力する
func Info(format string, args ...interface{}) {
	infoLogger.Printf(format, args...)
}

// Error はエラーメッセージを出力する
func Error(format string, args ...interface{}) {
	errorLogger.Output(2, fmt.Sprintf(format, args...))
}
