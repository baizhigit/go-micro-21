package main

import (
	"context"
	"fmt"
	"log"
	"log-service/data"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	webPort  = "8084"
	rpcPort  = "5001"
	mongoURL = "mongodb://mongo:27017"
	gRpcPort = "50001"
)

var client *mongo.Client

type Config struct {
	Models data.Models
}

func main() {
	// connect to mongo
	mongoClient, err := connectToMongo()
	if err != nil {
		log.Panic(err)
	}
	client = mongoClient
	log.Println("Connected to MongoDB!")

	// gracefully close connection when shutting down
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from Mongo: %v", err)
			panic(err)
		} else {
			log.Println("Disconnected from MongoDB")
		}
	}()

	app := Config{
		Models: data.New(client),
	}

	// --- here you can start HTTP or gRPC servers ---
	err = rpc.Register(new(RPCServer))
	if err != nil {
		panic(err)
	}
	go app.rpcListen()

	go app.gRPCListen()

	log.Println("Starting service on port: ", webPort)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	err = srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func (app *Config) rpcListen() {
	log.Println("Starting RPC server on port: ", rpcPort)
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", rpcPort))
	if err != nil {
		panic(err)
	}
	defer listen.Close()

	for {
		rpcConn, err := listen.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(rpcConn)
	}
}

func connectToMongo() (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(mongoURL).SetAuth(options.Credential{
		Username: "admin",
		Password: "admin",
	})

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Println("❌ Error connecting:", err)
		return nil, err
	}

	// Check connectivity (requires context)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		log.Println("⚠️ Mongo ping failed:", err)
		return nil, err
	}

	return client, nil
}
