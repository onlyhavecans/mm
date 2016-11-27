package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const base_dir string = "muck"
const in_file string = "in"
const out_file string = "out"

var (
	connection_name   string
	connection_server string
	connection_port   uint16
	use_ssl           bool
	debug_mode        bool
)

func debug_log(log ...string) {
	if debug_mode {
		fmt.Println(log)
	}
}

func get_timestamp() string {
	return time.Now().Format("2006-01-02T150405")
}

func init_vars() {
	flag.BoolVar(&use_ssl, "ssl", false, "Enable ssl")
	flag.BoolVar(&debug_mode, "debug", false, "Enable debug")
	flag.Parse()

	connection_args := flag.Args()
	if len(connection_args) != 3 {
		log.Fatal("Usage: mm [--ssl] [--debug] <name> <server> <port>")
	}
	connection_name = connection_args[0]
	connection_server = connection_args[1]
	if s, err := strconv.Atoi(connection_args[2]); err == nil {
		connection_port = uint16(s)
	} else {
		log.Fatal("Port must be a number 1 - 65535")
	}

	debug_log("Name:", connection_name)
	debug_log("Server:", connection_server)
	debug_log("Port:", strconv.Itoa(int(connection_port)))
	debug_log("SSL?:", strconv.FormatBool(use_ssl))
}

func get_working_dir(base string, connection string) string {
	var home string

	if dir, err := homedir.Dir(); err == nil {
		home = dir
		debug_log("Home directory", home)
	} else {
		log.Fatal("unable to expand home dir")
	}

	working := filepath.Join(home, base, connection)
	return working
}

func move_to_dir(directory string) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		log.Fatalf("Unable to make connection directory %v\n", directory)
	}

	if err := os.Chdir(directory); err != nil {
		log.Fatalf("Unable to chdir to %v\n", directory)
	}
}

func make_in(file string) {
	if _, err := os.Stat(file); err == nil {
		fmt.Println("FIFO already exists. Unlink or exit")
		fmt.Println("if you run multiple connection with the same name you're gonna have a bad time")
		fmt.Print("Type YES to unlink and recreate: ")
		input := bufio.NewReader(os.Stdin)
		if answer, err := input.ReadString('\n'); err != nil || answer != "YES" {
			log.Fatal("Canceling. Please remove FIFO before running")
		} else {
			syscall.Unlink(file)
		}
	}

	if err := syscall.Mkfifo(file, 0644); err != nil {
		log.Fatalf("Unable to make FIFO %v", file)
	}
}

func open_connection(server string) net.Conn {
	if conn, err := net.Dial("tcp", server); err != nil {
		log.Fatal("Unable to connect: ", err)
	} else {
		return connection
	}
}

func main() {
	fmt.Println("Started at", get_timestamp())
	init_vars()

	working_dir := get_working_dir(base_dir, connection_name)
	move_to_dir(working_dir)

	make_in(in_file)
	defer syscall.Unlink(in_file)

	//create connection with in_file to write and out_file to read
	connection_string := fmt.Sprintf("%s:%d", connection_server, connection_port)
	connection := open_connection()

	defer connection.Close()

	//defer rolling out

}
