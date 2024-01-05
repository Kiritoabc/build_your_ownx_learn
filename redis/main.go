package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

// 参考1：https://www.build-redis-from-scratch.dev/en/introduction
// 参考2：https://www.cnblogs.com/Finley/category/1598973.html

// create a listen and server --> IO 多路复用

func ListenAndServe(address string) {
	listen, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(fmt.Sprintf("listen err: %v", err))
	}
	defer listen.Accept()
	log.Println(fmt.Sprintf("bind: %s, start listening...", address))

	// load aof
	err = initOrLoadAof()
	if err != nil {
		log.Fatal(fmt.Sprintf("load aof err: %v", err))
	}
	defer AOF.Close()
	for {
		// Accept 会一直阻塞直到有新的连接建立或者listen中断才会返回
		conn, err := listen.Accept()
		if err != nil {
			// 通常是由于listener被关闭无法继续监听导致的错误
			log.Fatal(fmt.Sprintf("accept err: %v", err))
		}
		// start a goroutine to handle conn
		go Handle(conn)
	}
}

// Handle the connection
func Handle(conn net.Conn) {
	// use bufio to create a reader
	resp := NewResp(conn)
	// close connection
	defer conn.Close()
	for {
		value, err := resp.Read()
		if err != nil {
			fmt.Println(err)
			return
		}
		// if close
		if value.typ == "" {
			return
		}
		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]

		writer := NewWriter(conn)

		handlers, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command: ", command)
			writer.Write(Value{typ: "string", str: ""})
			continue
		}
		// 写入到aof
		if command == "SET" || command == "HSET" {
			AOF.Write(value)
		}
		result := handlers(args)
		writer.Write(result)
	}
}

var AOF *Aof

// load aof
func initOrLoadAof() (err error) {
	// load aof
	AOF, err = NewAof("database.aof")

	AOF.Read(func(value Value) {
		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]

		handler, ok := Handlers[command]
		if !ok {
			fmt.Println("Invalid command", command)
			return
		}
		handler(args)
	})
	return err
}

func main() {

	// start server
	ListenAndServe(":6379")
}
