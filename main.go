package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"

	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

const baseDir string = "muck"
const inFile string = "in"
const outFile string = "out"

var (
	connectionName   string
	connectionServer string
	connectionPort   uint
	useSSL           bool
	debugMode        bool
)

func debugLog(log ...string) {
	if debugMode {
		fmt.Println(log)
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}

func getTimestamp() string {
	return time.Now().Format("2006-01-02T150405")
}

func initVars() {
	flag.BoolVar(&useSSL, "ssl", false, "Enable ssl")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug")
	flag.Parse()

	args := flag.Args()
	if len(args) != 3 {
		fmt.Println("Usage: mm [--ssl] [--debug] <name> <server> <port>")
		os.Exit(1)
	}
	connectionName = args[0]
	connectionServer = args[1]
	port, err := strconv.Atoi(args[2])
	checkError(err)
	connectionPort = uint(port)

	debugLog("Name:", connectionName)
	debugLog("Server:", connectionServer)
	debugLog("Port:", strconv.Itoa(int(connectionPort)))
	debugLog("SSL?:", strconv.FormatBool(useSSL))
}

func getWorkingDir(base string, connection string) string {
	var home string

	home, err := homedir.Dir()
	checkError(err)
	debugLog("Home directory", home)

	working := filepath.Join(home, base, connection)
	return working
}

func makeInFIFO(file string) {
	if _, err := os.Stat(file); err == nil {
		fmt.Println("FIFO already exists. Unlink or exit")
		fmt.Println("if you run multiple connection with the same name you're gonna have a bad time")
		fmt.Print("Type YES to unlink and recreate: ")
		input := bufio.NewReader(os.Stdin)
		answer, err := input.ReadString('\n')
		checkError(err)
		if answer == "YES" {
			syscall.Unlink(file)
		} else {
			fmt.Println("Canceling. Please remove FIFO before running")
			os.Exit(1)
		}
	}

	err := syscall.Mkfifo(file, 0644)
	checkError(err)
}

func readToOutfile(conn net.TCPConn, file os.File) {

}

func main() {
	fmt.Println("~Started at", getTimestamp())
	initVars()

	// Make and move to working directory
	workingDir := getWorkingDir(baseDir, connectionName)
	errMk := os.MkdirAll(workingDir, 0755)
	checkError(errMk)

	errCh := os.Chdir(workingDir)
	checkError(errCh)

	// Make the in FIFO
	makeInFIFO(inFile)
	defer syscall.Unlink(inFile)

	//create connection with inFile to write and outFile to read
	connectionString := fmt.Sprintf("%s:%d", connectionServer, connectionPort)
	tcpAddress, errRes := net.ResolveTCPAddr("tcp4", connectionString)
	checkError(errRes)

	connection, errCon := net.DialTCP("tcp", nil, tcpAddress)
	checkError(errCon)
	fmt.Println("~Connected at", getTimestamp())
	defer connection.Close()

	// We keep alive for mucks
	errSka = connection.SetKeepAlive(true)
	checkError(errSka)
	var keepalive time.Duration = 15 * time.Minute
	errSkap = connection.SetKeepAlivePeriod(keepalive)

	out, errOut := os.Create(outFile)
	checkError(errOut)
	defer out.Close()

	go readToOutfile(connection, out)

	//defer rolling out

}
