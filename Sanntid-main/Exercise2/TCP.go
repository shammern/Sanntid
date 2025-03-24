package main

//Import necesary tools
import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {

	//conn, err := net.Dial("tcp", "10.100.23.204:34933") //Fixed size
	conn, err := net.Dial("tcp", "10.100.23.204:33546") // \0 delimiter
	if err != nil {
		fmt.Println("Error setting up connection:", err)
		os.Exit(1)
		return
	}

	defer conn.Close()

	//Setting up clientside server to accept connection from mainserver
	listener, err := net.Listen("tcp", "10.100.23.30:20020")
	if err != nil {
		fmt.Println("Error setting up listener:", err)
		os.Exit(1)
	}

	defer listener.Close()
	fmt.Println("Listening on 10.100.23.30:20020")
	
	data := make([]byte, 1024)
	message := "Connect to: 10.100.23.30:20020\000"
	copy(data,message)
	
	_, err = conn.Write(data)
	if err != nil{
		fmt.Println("Error sending data: ", err)
		return
	}

	buffer := make([]byte, 1024)

	for i:=0; i<1; i++{

		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		fmt.Printf("A connection has been established\n")

		n, err := conn.Read(buffer)
		if err != nil{
			fmt.Println("Error: ", err)
			return
		}
	
		//Trimming unfilled data from the buffer
		message := string(buffer[:n])
		trimmedMSG := strings.Trim(message, "\x00")

		fmt.Printf("Received: %s\n", trimmedMSG)
	}

	for i := 0; i < 10; i++{
		data := make([]byte, 1024)
		message := fmt.Sprintf("Sending data: %d\000", i)
	    copy(data,message)

		_, err = conn.Write(data)
		if err != nil{
			fmt.Println("Error sending data: ", err)
			return
		} 

		n, err := conn.Read(buffer)
		if err != nil{
			fmt.Println("Error: ", err)
			return
		}

		//Trimming unfilled data from the buffer
		RecMessage := string(buffer[:n])
		trimmedMSG := strings.Trim(RecMessage, "\x00")

		fmt.Printf("Received: %s\n", trimmedMSG)
		time.Sleep(100*time.Millisecond)
	}
}