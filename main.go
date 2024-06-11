package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

const logPath = "/var/log/ESF/eslogger.log"

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Not enough arguments.")
	}

	esloggerArgs := append([]string{"--format", "json"}, os.Args[1:]...)

	err := os.MkdirAll(filepath.Dir(logPath), os.ModePerm)
	if err != nil {
		log.Fatal("Unable to create /var/log/ESF: ", err)
	}

	cmd := exec.Command("/usr/bin/eslogger", esloggerArgs...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("Unable to create stdout pipe: ", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal("Unable to create stderr pipe: ", err)
	}

	reader := bufio.NewReader(stdoutPipe)
	writer := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100, // megabytes
		MaxBackups: 7,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	}

	defer writer.Close()

	err = cmd.Start()
	if err != nil {
		log.Fatalf("Unable to start cmd %v: %v", cmd.Args, err)
	}
	log.Printf("Process %s started with PID: %d\n", cmd.Args, cmd.Process.Pid)

	wg := sync.WaitGroup{}
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			log.Println("Process stderr: ", scanner.Text())
		}
	}()

	_, err = io.Copy(writer, reader)
	if err != nil {
		log.Fatal("Copy stdout failed: ", err)
	}

	state, err := cmd.Process.Wait()
	if err != nil {
		log.Fatal("Process wait error: ", err)
	}

	log.Printf("Process %s exit code: %d\n", cmd.Args, state.ExitCode())
}
