protoc -I=. --micro_out=./grpc/ --go_out=./grpc/ ./grpc/grpc.proto
move /Y .\grpc\github.com\paysuper\paysuper-webhook-notifier\pkg\proto\grpc\grpc.micro.go .\grpc\grpc.micro.go
move /Y .\grpc\github.com\paysuper\paysuper-webhook-notifier\pkg\proto\grpc\grpc.pb.go .\grpc\grpc.pb.go
rmdir /Q/S .\grpc\github.com
protoc-go-inject-tag -input=./grpc/grpc.pb.go -XXX_skip=bson,json,structure,validate