package logging

import (
	"log"
	"os"
)

func NewSessionLogger() (*log.Logger, func() error, error) {
	file, err := os.OpenFile("omron-mcp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, nil, err
	}
	return log.New(file, "", log.LstdFlags), file.Close, nil
}
