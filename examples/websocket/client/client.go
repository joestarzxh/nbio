package main

import (
	"flag"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:9001", "http service address")
var message = flag.String("message", "hello would", "message send to the server")
var messageLen = flag.Int("mlen", 100000, "if set, will override message setting and send message of the specified length")

func main() {
	flag.Parse()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	text := *message
	if *messageLen > 0 {
		textBytes := make([]byte, *messageLen)
		for i := 0; i < *messageLen; i++ {
			textBytes[i] = (*message)[i%(len(*message)-1)]
		}
		text = string(textBytes)
	}
	for {
		err := c.WriteMessage(websocket.TextMessage, []byte(text))
		if err != nil {
			log.Fatalf("write: %v", err)
			return
		}
		log.Println("write:", text)

		_, reader, err := c.NextReader()
		if err != nil {
			log.Println("read:", err)
			return
		}
		line := make([]byte, 1024)
		i := 0
		for err == nil {
			_, err = reader.Read(line)
			if err != nil {
				log.Println("read :", i, string(line))
			}
		}
		if err != io.EOF {
			log.Println("reader read error:", err)
			return
		}
		time.Sleep(time.Second)
	}
}
