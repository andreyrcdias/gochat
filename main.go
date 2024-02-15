package main

import (
	"log"
	"net"
	"time"
	"unicode/utf8"
)

const (
	Port = "4200"
)

type MessageType int

const (
	ClientConnected MessageType = iota + 1
	ClientDisconnected
	NewMessage
)

type Message struct {
	Type MessageType
	Conn net.Conn
	Text string
}

type Client struct {
	Conn        net.Conn
	LastMessage time.Time
}

func server(messages chan Message) {
	clients := map[string]*Client{}
	for {
		msg := <-messages
		switch msg.Type {
		case ClientConnected:
			addr := msg.Conn.RemoteAddr().(*net.TCPAddr)
			log.Printf("Client %s connected", addr.String())
			clients[msg.Conn.RemoteAddr().String()] = &Client{
				Conn:        msg.Conn,
				LastMessage: time.Now(),
			}
		case ClientDisconnected:
			addr := msg.Conn.RemoteAddr().(*net.TCPAddr)
			log.Printf("Client %s disconnected", addr.String())
			delete(clients, addr.String())
		case NewMessage:
			authorAddr := msg.Conn.RemoteAddr().(*net.TCPAddr)
			author := clients[authorAddr.String()]
			now := time.Now()
			if author != nil {
				if utf8.ValidString(msg.Text) {
					author.LastMessage = now
					log.Printf("Client %s sent message %s", authorAddr.String(), msg.Text)
					for _, client := range clients {
						if client.Conn.RemoteAddr().String() != authorAddr.String() {
							client.Conn.Write([]byte(msg.Text))
						}
					}
				}
			} else {
				msg.Conn.Close()
			}
		}
	}
}

func client(conn net.Conn, messages chan Message) {
	buffer := make([]byte, 64)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			conn.Close()
			messages <- Message{
				Type: ClientDisconnected,
				Conn: conn,
			}
			return
		}
		text := string(buffer[0:n])
		messages <- Message{
			Type: NewMessage,
			Text: text,
			Conn: conn,
		}
	}

}

func main() {
	ln, err := net.Listen("tcp", ":"+Port)
	if err != nil {
		log.Fatalf("Cloud not listen to port %s: %s\n", Port, err.Error())

	}
	log.Printf("Listening to TCP connections on port %s...\n", Port)

	messages := make(chan Message)
	go server(messages)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Cloud not accept a connection: %s\n", err.Error())
			continue
		}
		messages <- Message{
			Type: ClientConnected,
			Conn: conn,
		}
		go client(conn, messages)
	}
}
