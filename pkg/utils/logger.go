package utils

import (
	"log"
	"os"
)

// logger는 utils 패키지 전체에서 사용될 로거 인스턴스입니다.
var logger *log.Logger

func init() {
	// init 함수는 패키지가 로드될 때 자동으로 호출됩니다.
	// 여기서 로거를 초기화하여 [FAVUS] 접두사, 날짜/시간, 파일/라인 정보를 포함하도록 설정합니다.
	logger = log.New(os.Stdout, "[FAVUS] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Info logs an info message.
func Info(format string, v ...interface{}) {
	logger.Printf("INFO: "+format, v...)
}

// Error logs an error message.
func Error(format string, v ...interface{}) {
	logger.Printf("ERROR: "+format, v...)
}

// Fatal logs a fatal message and exits the program.
// Critical errors that prevent further operation should use Fatal.
func Fatal(format string, v ...interface{}) {
	logger.Fatalf("FATAL: "+format, v...)
}

// NewLogger 함수는 이 접근 방식에서는 더 이상 필요하지 않습니다.
// 하지만 만약 필요하다면 다음과 같이 더미 함수로 남겨둘 수 있습니다.
// 혹은 이 함수 자체가 제거되어야 합니다.
// func NewLogger() *Logger {
//     return &Logger{logger} // 만약 Logger struct를 유지한다면
// }
