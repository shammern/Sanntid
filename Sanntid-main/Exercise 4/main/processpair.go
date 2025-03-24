package main

import (
	"fmt"
	"net"
	"os"
	"os/exec" //for å kunne åpne nytt vindu
	"strconv"
	"time"
)

const (
	udpIP             = "127.0.0.1"
	udpPort           = 5005
	heartbeatInterval = 1 * time.Second
	timeoutDuration   = 3 * time.Second
)

func spawnBackup() {
	fmt.Println("Starting backup-process...")
	exe, err := os.Executable() //returnerer sti til kjørende program
	if err != nil {
		fmt.Println("Error when retriving file:", err)
		return
	}
	//Windows bruker vi cmd til å starte en ny terminal, exe er hentet sti
	cmd := exec.Command("cmd", "/C", "start", "Ny teller", exe, "backup")
	err = cmd.Start() //Start returnerer dersom den ikke får til å åpne nytt vindu
	if err != nil {
		fmt.Println("Feil ved oppstart av backup:", err)
	}
}

func runPrimary(startCounter int) {
	counter := startCounter //telling på gang
	backupSpawned := false  //backupen skal kun starte 1 gang
	for {
		fmt.Println(counter)

		//Sender heartbeat via UDP
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", udpIP, udpPort)) //oppretter addresse
		if err != nil {
			fmt.Println("Feil ved oppsett av UDP-adresse:", err)
		}
		conn, err := net.DialUDP("udp", nil, addr) //oppretter forbindelse for å kunne sende data
		if err != nil {
			fmt.Println("Feil ved tilkobling til UDP:", err)
		}
		message := []byte(strconv.Itoa(counter)) //konvertere tall til streng for å sende
		if conn != nil {
			conn.Write(message) //skriver mld
			conn.Close()        //deretter lukker tilkoblingen
		}

		counter++
		time.Sleep(heartbeatInterval) //går ett sekund

		//Start backup én gang
		if !backupSpawned { //hvis ikke, spawn og sett lik bool true
			spawnBackup()
			backupSpawned = true
		}
	}
}

func runBackup() {
	fmt.Println("Backup-process started, overvåker primær...")
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", udpIP, udpPort)) // setter opp addresse
	if err != nil {
		fmt.Println("Feil ved oppsett av UDP-adresse:", err)
		return
	}
	conn, err := net.ListenUDP("udp", addr) //lytter på kanalen
	if err != nil {
		fmt.Println("Feil ved lytting på UDP-port:", err)
		return
	}
	defer conn.Close() //lukker forbindelse når funksjonen er ferdig, defer skjer når funksjonen rundt avsluttes

	lastCounter := 0          //huske siste motatt tall
	buf := make([]byte, 1024) //leser data fra UDP
	//Setter første timeout, her 3 sek for å lese heartbeat
	conn.SetReadDeadline(time.Now().Add(timeoutDuration)) //leser data inn i buff (pågående tallet)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			// Sjekk om feilen er en timeout
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("Primær ikke oppdaget. Tar over som primær.")
				conn.Close()                //lukk UDP-forbindelsen før vi tar over
				spawnBackup()               //starter en ny backup før vi tar over
				runPrimary(lastCounter + 1) //starter ny telling fra siste tall
				return
			} else {
				fmt.Println("Feil ved mottak av heartbeat:", err)
				time.Sleep(heartbeatInterval)
			}
		} else {
			s := string(buf[:n])
			lastCounter, err = strconv.Atoi(s) //koverterer motatt beskjed til tall
			if err != nil {
				fmt.Println("Feil ved konvertering av heartbeat:", err)
			}
			//Nullstiller timeout hver gang vi får heartbeat
			conn.SetReadDeadline(time.Now().Add(timeoutDuration))
		}
	}
}

func main() {
	role := "primary" //rolle
	if len(os.Args) > 1 {
		role = os.Args[1] //dersom os inneholder 'backup' på slutten, endre rollen
	}
	//Velger her utifra rolle
	if role == "primary" {
		runPrimary(1) //starter på 1
	} else if role == "backup" {
		runBackup()
	} else {
		fmt.Println("Ukjent rolle. Bruk 'primary' eller 'backup'.")
	}
}
