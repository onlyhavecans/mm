package main

import (
	"bufio"
	"crypto/tls"
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

// hardcoded program settings
const baseDir string = "muck"
const inFile string = "in"
const outFile string = "out"
const timeString string = "2006-01-02T150405"

// Program level switches set from command line
var debugMode bool
var disableLogRotate bool

// MuckServer stores all connection settings
type MuckServer struct {
	name     string
	host     string
	port     uint
	ssl      bool
	insecure bool
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
		panic(err.Error())
	}
}

func getTimestamp() string {
	return time.Now().Format(timeString)
}

func initArgs() MuckServer {
	flag.BoolVar(&debugMode, "debug", false, "Enable debug")
	flag.BoolVar(&disableLogRotate, "nolog", false, "Disable log rotation on quit")
	ssl := flag.Bool("ssl", false, "Enable ssl")
	insecure := flag.Bool("insecure", false, "Disable strict SSL checking")
	flag.Parse()

	args := flag.Args()
	if len(args) != 3 {
		fmt.Println("Usage: mm [--debug] [--nolog] [--ssl] [--insecure] <name> <server> <port>")
		os.Exit(1)
	}
	p, err := strconv.Atoi(args[2])
	checkError(err)

	s := MuckServer{name: args[0], host: args[1], port: uint(p), ssl: *ssl, insecure: *insecure}

	debugLog("rotate log disabled?:", disableLogRotate)
	debugLog("name:", s.name)
	debugLog("host:", s.host)
	debugLog("port:", s.port)
	debugLog("SSL?:", s.ssl)
	debugLog("insecure ssl check?:", s.insecure)

	return s
}

func getWorkingDir(main string, sub string) string {
	u, err := user.Current()
	checkError(err)
	h := u.HomeDir
	debugLog("home directory", h)

	w := filepath.Join(h, main, sub)
	debugLog("working directory", w)
	return w
}

func makeFIFO(file string) *os.File {
	if _, err := os.Stat(file); err == nil {
		fmt.Println("FIFO already exists. Unlink or exit")
		fmt.Println("If you run multiple connection with the same name you're gonna have a bad time")
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
	f, err := os.OpenFile(file, os.O_RDONLY|syscall.O_NONBLOCK, 0666)
	checkError(err)
	debugLog("FIFO opened as", f.Name())
	return f
}

func makeOut(file string) *os.File {
	if _, err := os.Stat(file); err == nil {
		fmt.Printf("Warning: %v already exists; appending.\n", file)
	}
	out, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	checkError(err)
	debugLog("logfile created as", out.Name())
	return out
}

func lookupHostname(s string) *net.TCPAddr {
	a, err := net.ResolveTCPAddr("tcp", s)
	checkError(err)
	debugLog("server resolves to", a)
	return a
}

func setupConnection(s *MuckServer) net.Conn {
	tcpAddr := lookupHostname(s.String())
	connection, err := net.DialTCP("tcp", nil, tcpAddr)
	checkError(err)
	debugLog("connected to Server")

	// We keep alive for mucks
	errSka := connection.SetKeepAlive(true)
	checkError(errSka)
	keepalive := 15 * time.Minute
	errSkap := connection.SetKeepAlivePeriod(keepalive)
	checkError(errSkap)
	return connection
}

func setupTLSConnextion(s *MuckServer) net.Conn {
	var conf *tls.Config
	if s.insecure == true {
		conf = &tls.Config{InsecureSkipVerify: true}
	} else {
		conf = &tls.Config{ServerName: s.host}
	}
	tcpAddr := lookupHostname(s.String())
	connection, err := tls.Dial("tcp", tcpAddr.String(), conf)
	checkError(err)
	return connection
}

func readtoConn(f *os.File, c net.Conn, quit chan bool) {
	tmpError := fmt.Sprintf("read %v: resource temporarily unavailable", f.Name())
	for {
		select {
		case <-quit:
			debugLog("readtoConn recieved quit; returning")
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
	_, err := f.WriteString(fmt.Sprintf("~Connected at %v\n", getTimestamp()))
	checkError(err)
	for {
		buf := make([]byte, 512)
		bi, err := c.Read(buf)
		if err != nil {
			fmt.Println("Server disconnected with", err.Error())
			_, err := f.WriteString(fmt.Sprintf("\n~Connection lost at %v\n", getTimestamp()))
			checkError(err)
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
	err := c.Close()
	if err != nil {
		debugLog(err.Error())
	}
	debugLog("connection closed")
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
	debugLog(n, "closed")
	if disableLogRotate == true {
		debugLog("log rotation is disabled")
		return
	}
	errR := os.Rename(outFile, getTimestamp())
	if errR != nil {
		debugLog(errR.Error())
	}
	debugLog(n, "rotated")
}

func main() {
	// checkError throws a panic, catch it at the end and return error to user
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("FATAL ERROR", r)
		}
	}()

	fmt.Println("Started at", getTimestamp())
	server := initArgs()

	// Make and move to working directory
	workingDir := getWorkingDir(baseDir, server.name)
	errMk := os.MkdirAll(workingDir, 0755)
	checkError(errMk)

	errCh := os.Chdir(workingDir)
	checkError(errCh)

	// Make the in FIFO
	in := makeFIFO(inFile)
	defer closeFIFO(in)

	// Make the out file
	out := makeOut(outFile)
	defer closeLog(out)

	//create connection
	var connection net.Conn
	if server.ssl == true {
		connection = setupTLSConnextion(&server)
	} else {
		connection = setupConnection(&server)
	}
	defer closeConnection(connection)

	quit := make(chan bool)
	go readToFile(connection, out, quit)
	readtoConn(in, connection, quit)

	fmt.Println("Quit at", getTimestamp())
	fmt.Println("Thanks for playing!")
	debugLog("end of main hit")
}
