package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	pb "github.com/NukeDev/Goolia/proto"

	"google.golang.org/grpc"
)

type server struct{}
type CommandLine struct{}

type Client struct {
	ID       string
	ClientIP string
	LastPing time.Time
}

var Clients = map[string]Client{}
var Command = map[string]string{}

func (s server) HandleCommands(srv pb.Com_HandleCommandsServer) error {

	log.Println("Server Started")

	ctx := srv.Context()

	for {

		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// receive data from stream
		req, err := srv.Recv()

		if err == io.EOF {
			// return will close stream from server side
			log.Println("EOF")
			return nil
		}
		if err != nil {
			log.Printf("receive error %v", err)
			continue
		}

		// continue if number reveived from stream
		// less than max

		switch req.Command {
		case "ping":
			{
				cl := Client{
					ID:       req.ClientID,
					ClientIP: req.ClientIPAddress,
					LastPing: time.Now(),
				}

				Clients[cl.ID] = cl

				resp := pb.Response{ClientID: cl.ID, ClientIPAddress: cl.ClientIP, Command: "ping", Data: nil}

				if err := srv.Send(&resp); err != nil {
					log.Printf("PING: error %v", err)
				}

			}
		default:
			{
				resp := pb.Response{ClientID: req.ClientID, ClientIPAddress: req.ClientIPAddress, Command: "not-found", Data: nil}
				if err := srv.Send(&resp); err != nil {
					log.Printf("Command Not-Found error %v", err)
				}
				log.Printf("Sent command not found to client id=%s", req.ClientID)
			}
		}

	}
}

func main() {
	// create listiner
	go func() {
		lis, err := net.Listen("tcp", ":50005")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		// create grpc server
		s := grpc.NewServer()
		pb.RegisterComServer(s, server{})

		// and start...
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	cmd := CommandLine{}
	cmd.Run()

}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("clients - Gets connected clients")
	fmt.Println("osinfo -clientid - Gets OSINFO of specified client")
}

func (cli *CommandLine) validateArgs(data []string) {
	if len(data) < 1 {
		cli.printUsage()
	}
}

func (cli *CommandLine) getClients() {
	keys := make([]string, 0, len(Clients))
	for k := range Clients {
		keys = append(keys, k)
	}

	for x := range keys {
		log.Println(x)
	}
}

func (cli *CommandLine) Run() {

	for {

		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		data := strings.Fields(input.Text())
		cli.validateArgs(data)

		clients := flag.NewFlagSet("clients", flag.ExitOnError)
		osinfo := flag.NewFlagSet("osinfo", flag.ExitOnError)

		clientosinfo := osinfo.String("clientid", "", "Gets OSINFO of specified client")

		switch data[0] {
		case "clients":
			err := clients.Parse(data[1:])
			if err != nil {
				log.Panic(err)
			}
		case "osinfo":
			err := osinfo.Parse(data[1:])
			if err != nil {
				log.Panic(err)
			}
		default:
			cli.printUsage()
			runtime.Goexit()
		}

		if clients.Parsed() {

			cli.getClients()
		}

		if osinfo.Parsed() {
			if *clientosinfo == "" {
				osinfo.Usage()
			} else {
				time.Sleep(time.Second * 1)
			}

		}
	}

}
