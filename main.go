package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// mostly hardcoded program settings
const baseDir string = "muck"
const inFile string = "in"
const outFile string = "out"

// Keep debug global
var debugMode bool

// Simple struct for our connection
type MuckServer struct {
	name string
	host string
	port uint
	ssl  bool
}

// Simplify returning connection strings by making it the String method
func (m *MuckServer) String() string {
	s := fmt.Sprintf("%s:%d", m.host, m.port)
	return s
}

func debugLog(log ...interface{}) {
	if debugMode {
		fmt.Print("DEBUG: ")
		fmt.Println(log...)
	}
}

func checkError(err error) {
	if err != nil {
		debugLog("checkError caught", err.Error())
		s := fmt.Sprintln("fatal error", err.Error())
		panic(s)
	}
}

func getTimestamp() string {
	return time.Now().Format("2006-01-02T150405")
}

func initArgs() MuckServer {
	flag.BoolVar(&debugMode, "debug", false, "Enable debug")
	ssl := flag.Bool("ssl", false, "Enable ssl")
	flag.Parse()

	args := flag.Args()
	if len(args) != 3 {
		fmt.Println("Usage: mm [--ssl] [--debug] <name> <server> <port>")
		os.Exit(1)
	}
	p, err := strconv.Atoi(args[2])
	checkError(err)

	s := MuckServer{name: args[0], host: args[1], port: uint(p), ssl: *ssl}

	debugLog("name:", s.name)
	debugLog("host:", s.host)
	debugLog("port:", s.port)
	debugLog("SSL?:", s.ssl)

	return s
}

func getWorkingDir(main string, sub string) string {
	u, err := user.Current()
	checkError(err)
	h := u.HomeDir
	debugLog("Home directory", h)

	w := filepath.Join(h, main, sub)
	debugLog("working directory", w)
	return w
}

func setupConnection(s *MuckServer) net.Conn {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", s.String())
	checkError(err)
	debugLog("server resolves to", tcpAddr)
	connection, err := net.DialTCP("tcp", nil, tcpAddr)
	checkError(err)
	fmt.Println("~Connected at", getTimestamp())

	// We keep alive for mucks
	errSka := connection.SetKeepAlive(true)
	checkError(errSka)
	keepalive := 15 * time.Minute
	errSkap := connection.SetKeepAlivePeriod(keepalive)
	checkError(errSkap)
	return connection
}

func setupTLSConnextion(s *MuckServer) net.Conn {
	var connection net.Conn
	return connection

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
			panic("User Canceled at FIFO removal prompt")
		}
		errUn := syscall.Unlink(file)
		checkError(errUn)
		debugLog(file, "unlinked")
	}

	err := syscall.Mkfifo(file, 0644)
	checkError(err)
	debugLog("FIFO created as", file)
	f, err := os.OpenFile(file, syscall.O_RDONLY|syscall.O_NONBLOCK, 0666)
	checkError(err)
	debugLog("FIFO opened as", f.Name())
	return f
}

func readtoConn(f *os.File, c net.Conn, quit chan bool) {
	tmpError := fmt.Sprintf("read %v: resource temporarily unavailable", f.Name())
	for {
		select {
		case <-quit:
			debugLog("readtoConn got quit, returning")
			return
		default:
			buf := make([]byte, 512)
			bi, err := f.Read(buf)
			if err != nil && err.Error() != "EOF" && err.Error() != tmpError {
				checkError(err)
			} else if bi == 0 {
				continue
			}
			debugLog(bi, "bytes read from FIFO")
			bo, err := c.Write(buf[:bi])
			checkError(err)
			debugLog(bo, "bytes written to file")
		}
	}
}

func readToFile(c net.Conn, f *os.File, quit chan bool) {
	for {
		buf := make([]byte, 512)
		bi, err := c.Read(buf)
		if err != nil {
			fmt.Println("Connection broken,", err.Error())
			quit <- true
			return
		}
		debugLog(bi, "bytes read from connection")

		bo, err := f.Write(buf[:bi])
		checkError(err)
		debugLog(bo, "bytes written to file")
	}
}

func closeConnection(c net.Conn) {
	fmt.Println("~Closing connection at", getTimestamp())
	err := c.Close()
	if err != nil {
		debugLog(err.Error())
	}
	debugLog("Connection closed")
}

func closeFIFO(f *os.File) {
	n := f.Name()
	debugLog("closing and deleting FIFO", n)
	errC := f.Close()
	if errC != nil {
		debugLog(errC.Error())
	}
	errU := syscall.Unlink(n)
	if errU != nil {
		debugLog(errU.Error())
	}
	debugLog(n, "closed and deleted")
}

func closeLog(f *os.File) {
	n := f.Name()
	debugLog("closing and rotating file", n)
	errC := f.Close()
	if errC != nil {
		debugLog(errC.Error())
	}
	if _, err := os.Stat(n); err != nil {
		fmt.Println("out file doesn't exist? not rotating")
		return
	}
	errR := os.Rename(outFile, getTimestamp())
	if errR != nil {
		debugLog(errR.Error())
	}
	debugLog(n, "closed and rotated")
}

func main() {
	// checkError throws a panic, catch it at the end and return error to user
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("FATAL ERROR", r)
		}
	}()

	fmt.Println("~Started at", getTimestamp())
	server := initArgs()

	// Make and move to working directory
	workingDir := getWorkingDir(baseDir, server.name)
	errMk := os.MkdirAll(workingDir, 0755)
	checkError(errMk)

	errCh := os.Chdir(workingDir)
	checkError(errCh)

	//create connection
	var connection net.Conn
	if server.ssl == true {
		connection = setupTLSConnextion(&server)
	} else {
		connection = setupConnection(&server)
	}
	defer closeConnection(connection)

	// Make the in FIFO
	in := makeFIFO(inFile)
	defer closeFIFO(in)

	// Make the out file
	out, err := os.Create(outFile)
	checkError(err)
	debugLog("Logfile created as", out.Name())
	defer closeLog(out)

	quit := make(chan bool)
	go readToFile(connection, out, quit)
	readtoConn(in, connection, quit)

	debugLog("End of main hit")
}
