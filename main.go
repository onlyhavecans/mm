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

func setupConnection(s string) *net.TCPConn {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", s)
	checkError(err)
	debugLog("server resolves to", tcpAddr)
	connection, err := net.DialTCP("tcp", nil, tcpAddr)
	checkError(err)
	fmt.Println("~Connected at", getTimestamp())

	// We keep alive for mucks
	errSka := connection.SetKeepAlive(true)
	checkError(errSka)
	var keepalive time.Duration = 15 * time.Minute
	errSkap := connection.SetKeepAlivePeriod(keepalive)
	checkError(errSkap)
	return connection
}

func closeConnection(c *net.TCPConn) {
	fmt.Println("~Closing connection at", getTimestamp())
	c.Close()
}

func makeFIFO(file string) *os.File {
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
	f, err := os.Open(file)
	checkError(err)
	return f
}

func closeFIFO(f *os.File) {
	n := f.Name()
	f.Close()
	syscall.Unlink(n)
}

func closeLog(f *os.File) {
	f.Close()
	err := os.Rename(outFile, getTimestamp())
	checkError(err)
}

func readtoConn(f *os.File, c *net.TCPConn) {
	for {
		buf := make([]byte, 512)
		bi, err := f.Read(buf)
		checkError(err)
		debugLog(bi, "bytes read from FIFO")
		bo, err := c.Write(buf[:bi])
		checkError(err)
		debugLog(bo, "bytes written to file")
	}
}

func readToFile(c *net.TCPConn, f *os.File) {
	for {
		buf := make([]byte, 512)
		bi, err := c.Read(buf)
		if err != nil {
			continue
		}
		debugLog(bi, "bytes read from connection")

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

	//create connection
	server := fmt.Sprintf("%s:%d", connectionServer, connectionPort)
	connection := setupConnection(server)
	defer closeConnection(connection)

	// Make the in FIFO
	in := makeFIFO(inFile)
	defer closeFIFO(in)

	// Make the out file
	out, err := os.Create(outFile)
	checkError(err)
	defer closeLog(out)

	go readtoConn(in, connection)
	readToFile(connection, out)
}
