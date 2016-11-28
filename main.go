package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"net"
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
	connection_port   uint
	use_ssl           bool
	debug_mode        bool
)

func debug_log(log ...string) {
	if debug_mode {
		fmt.Println(log)
	}
}

func check_error(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
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
		fmt.Println("Usage: mm [--ssl] [--debug] <name> <server> <port>")
		os.Exit(1)
	}
	connection_name = connection_args[0]
	connection_server = connection_args[1]
	s, err := strconv.Atoi(connection_args[2])
	check_error(err)
	connection_port = uint(s)

	debug_log("Name:", connection_name)
	debug_log("Server:", connection_server)
	debug_log("Port:", strconv.Itoa(int(connection_port)))
	debug_log("SSL?:", strconv.FormatBool(use_ssl))
}

func get_working_dir(base string, connection string) string {
	var home string

	home, err := homedir.Dir()
	check_error(err)
	debug_log("Home directory", home)

	working := filepath.Join(home, base, connection)
	return working
}

func make_in(file string) {
	if _, err := os.Stat(file); err == nil {
		fmt.Println("FIFO already exists. Unlink or exit")
		fmt.Println("if you run multiple connection with the same name you're gonna have a bad time")
		fmt.Print("Type YES to unlink and recreate: ")
		input := bufio.NewReader(os.Stdin)
		answer, err := input.ReadString('\n')
		check_error(err)
		if answer == "YES" {
			syscall.Unlink(file)
		} else {
			fmt.Println("Canceling. Please remove FIFO before running")
			os.Exit(1)
		}
	}

	err := syscall.Mkfifo(file, 0644)
	check_error(err)
}

func main() {
	fmt.Println("~Started at", get_timestamp())
	init_vars()

	// Make and move to working directory
	working_dir := get_working_dir(base_dir, connection_name)
	errMk := os.MkdirAll(working_dir, 0755)
	check_error(errMk)

	errCh := os.Chdir(working_dir)
	check_error(errCh)

	// Make the in FIFO
	make_in(in_file)
	defer syscall.Unlink(in_file)

	//create connection with in_file to write and out_file to read
	connection_string := fmt.Sprintf("%s:%d", connection_server, connection_port)
	connection, errCon := net.Dial("tcp", connection_string)
	check_error(errCon)
	fmt.Println("~Connected at", get_timestamp())
	defer connection.Close()

	out, errOut := os.Create(out_file)
	check_error(errOut)
	defer out.Close()

	go read_to_outfile(connection, out)

	//defer rolling out

}
