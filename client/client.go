package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"time"

	osinfo "github.com/NukeDev/Goolia/client/osinfo"
	pb "github.com/NukeDev/Goolia/proto"
	externalip "github.com/glendc/go-external-ip"

	"google.golang.org/grpc"
)

type Client struct {
	ID        string
	IPAddress string
}

var localClient Client
var commandWaitingList []string

func main() {

	// dail server
	conn, err := grpc.Dial(":50005", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}

	// create stream
	localClient.Generate()
	client := pb.NewComClient(conn)
	stream, err := client.HandleCommands(context.Background())
	if err != nil {
		log.Fatalf("openn stream error %v", err)
	}

	ctx := stream.Context()
	done := make(chan bool)

	// first goroutine sends random increasing numbers to stream
	// and closes int after 10 iterations
	go func() {
		for {

			// generate random nummber and send it to stream
			req := pb.Request{ClientID: localClient.ID, ClientIPAddress: localClient.IPAddress, Command: "ping", Data: nil}
			if err := stream.Send(&req); err != nil {
				log.Fatalf("can not send ping to master server %v", err)
			}
			log.Printf("Sending ping to master server")
			time.Sleep(time.Second * 10)
		}

	}()

	// second goroutine receives data from stream
	// and saves result in max variable
	//
	// if stream is finished it closes done channel
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Fatalf("can not receive %v", err)
			}

			switch resp.Command {
			case "ping":
				{
					log.Println("Ping OK!")
				}
			case "osinfo":
				{

					commandWaitingList = append(commandWaitingList, "osinfo")
					info, err := osinfo.GetOsInfo()
					if err != nil {
						continue
					}

					b, err := json.Marshal(info)

					if err != nil {
						log.Fatalf("%v", err)
					}

					req := pb.Request{ClientID: localClient.ID, ClientIPAddress: localClient.IPAddress, Command: "osinfo", Data: b}

					if err := stream.Send(&req); err != nil {
						log.Fatalf("can not send OSINFO to master server %v", err)
					}
					log.Printf("Sending OSINFO to master server")

				}
			case "not-found":
			default:
				{
					log.Println("No command found as %s", resp.Command)
				}
			}
		}
	}()

	// third goroutine closes done channel
	// if context is done
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		close(done)
	}()

	<-done
	log.Printf("finished with")
}

func (cl *Client) Generate() {
	b := make([]byte, 22)
	if _, err := rand.Read(b); err != nil {

	}
	cl.ID = hex.EncodeToString(b)

	consensus := externalip.DefaultConsensus(nil, nil)

	ip, err := consensus.ExternalIP()
	if err == nil {
		cl.IPAddress = (ip.String())
	} else {
		cl.IPAddress = "undefined"
	}
}
