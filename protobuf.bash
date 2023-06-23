protoc --go_out=plugins=grpc:. *.proto 
cd ./proto && go mod init github.com/NukeDev/Goolia-Rat/proto