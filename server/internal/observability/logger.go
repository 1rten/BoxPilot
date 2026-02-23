package observability

import "log"

// Logger wraps structured logging. For MVP use log.Printf.
func Info(msg string, kvs ...any) {
	log.Println("[INFO]", msg, kvs)
}

func Error(msg string, kvs ...any) {
	log.Println("[ERROR]", msg, kvs)
}
