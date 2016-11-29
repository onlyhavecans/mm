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

func debugLog(log ...interface{}) {
	if debugMode {
		fmt.Println(log)
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println("fatal error", err.Error())
		os.Exit(3162)
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
	p, err := strconv.Atoi(args[2])
	checkError(err)
	connectionPort = uint(p)

	debugLog("Name:", connectionName)
	debugLog("Server:", connectionServer)
	debugLog("Port:", connectionPort)
	debugLog("SSL?:", useSSL)
}

func getWorkingDir(main string, sub string) string {
	h, err := homedir.Dir()
	checkError(err)
	debugLog("Home directory", h)

	w := filepath.Join(h, main, sub)
	return w
}

func makeInFIFO(file string) {
	if _, err := os.Stat(file); err == nil {
		fmt.Println("FIFO already exists. Unlink or exit")
		fmt.Println("if you run multiple connection with the same name you're gonna have a bad time")
		fmt.Print("Type YES to unlink and recreate: ")
		i := bufio.NewReader(os.Stdin)
		a, err := i.ReadString('\n')
		checkError(err)
		if a != "YES\n" {
			fmt.Println("Canceling. Please remove FIFO before running")
			os.Exit(1)
		}
		syscall.Unlink(file)
	}

	err := syscall.Mkfifo(file, 0644)
	checkError(err)
}

func closeAndRollLog(f *os.File) {
	f.Close()
	err := os.Rename(outFile, getTimestamp())
	checkError(err)
}

func readToFile(c *net.TCPConn, f *os.File) {
	for {
		buf := make([]byte, 512)
		bi, err := c.Read(buf)
		checkError(err)
		debugLog(bi, "Bytes read from connection")

		bo, err := f.Write(buf[:bi])
		checkError(err)
		debugLog(bo, "bytes written to file")
	}
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
	connStr := fmt.Sprintf("%s:%d", connectionServer, connectionPort)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", connStr)
	checkError(err)
	debugLog("server resolves to", tcpAddr)
	connection, err := net.DialTCP("tcp", nil, tcpAddr)
	checkError(err)
	fmt.Println("~Connected at", getTimestamp())
	defer connection.Close()

	// We keep alive for mucks
	errSka := connection.SetKeepAlive(true)
	checkError(errSka)
	var keepalive time.Duration = 15 * time.Minute
	errSkap := connection.SetKeepAlivePeriod(keepalive)
	checkError(errSkap)

	out, err := os.Create(outFile)
	checkError(err)
	defer closeAndRollLog(out)

	readToFile(connection, out)

}
