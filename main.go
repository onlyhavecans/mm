package main

import (
	"flag"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"os"
	"strconv"
)

const main_dir string = "~/muck/"

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

func init_vars() {
	flag.BoolVar(&use_ssl, "ssl", false, "Enable ssl")
	flag.BoolVar(&debug_mode, "debug", false, "Enable debug")
	flag.Parse()

	connection_args := flag.Args()
	if len(connection_args) != 3 {
		fmt.Println("Usage: mm [--ssl] [--debug] <name> <server> <port>\n")
		os.Exit(5)
	}
	connection_name = connection_args[0]
	connection_server = connection_args[1]
	if s, err := strconv.Atoi(connection_args[2]); err == nil {
		connection_port = uint16(s)
	} else {
		fmt.Println("Port must be a number 1 - 65535")
		os.Exit(5)
	}

	debug_log("Name:", connection_name)
	debug_log("Server:", connection_server)
	debug_log("Port:", strconv.Itoa(int(connection_port)))
	debug_log("SSL?:", strconv.FormatBool(use_ssl))
}

func get_muck_dir() string {
	var muck_dir string
	if dir, err := homedir.Expand(main_dir); err == nil {
		muck_dir = dir
	} else {
		fmt.Println("unable to expand home dir")
		os.Exit(10)
	}
	debug_log("muckdir", muck_dir)

	return muck_dir
}

func move_to_main_directory() {
	muck_dir := get_muck_dir()
	if _, err := os.Stat(muck_dir); os.IsNotExist(err) {
		if err := os.Mkdir(muck_dir); err != nil {
			fmt.Printf("Unable to make non-existant dir %v\n", muck_dir)
			os.Exit(10)
		}
	}
	if err := os.Chdir(muck_dir); err != nil {
		fmt.Printf("Unable to chdir to %v\n", muck_dir)
		os.Exit(10)
	}
}

func main() {
	//set up settings
	init_vars()

	move_to_main_directory()
	//defer clean up connection, in, and roll out

	//create connection

	//create in
}
