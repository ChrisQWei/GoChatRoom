package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var clients = make(map[*websocket.Conn]bool) //Connected clients
//This variable is a map where the key is actually a pointer to a WebSocket.
//The value is just a boolean. The value isn't actually needed but we
//are using a map because it is easier than an array to append and delete items.

var broadcast = make(chan Message) //broadcast channel
//This variable is a channel that will act as a queue for messages sent by clients.

var upgrader = websocket.Upgrader{} //Configure the upgrader
//Here create an instance of an upgrader. This is just an object with methods for taking a
//normal HTTP connection and upgrading it to a WebSocket.

type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
} //Define our message object
//Here we defined an object to hold our messages. It's simple struct with some string attributes
//for an email address, a username and the actual message. We used the email to display a unique
//avatar provided by the popular Gravatar service.

func main() {
	//Create a simple file server
	fs := http.FileServer(http.Dir("../public"))
	http.Handle("/", fs)
	//Here we create a static fileserver and tie that to the "/" route so that when a user accesses
	//the site they will be able to view index.html and any assets.

	//Configure websocket route
	http.HandleFunc("/ws", handleConnections)
	//we defined "/ws" which is where we will handle any requests for initiating a WebSocket

	//Start listening for incoming chat messages
	go handleMessages()
	//We started a goroutine called "handleMessages". This is a concurrent process that will
	//run along side the rest of the application that only take messages from the broadcast channel
	//from before and the pass them to clients over their respective Websocket connection.

	//Start the server on localhost port 8000 and log any errors
	log.Println("Http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	//Here we print a helpful message to the console and start the webserver.
	//If there are any errors we log them and exit the application.
}

//We need to create the function to handle our incoming WebSocket connections.
func handleConnections(w http.ResponseWriter, r *http.Request) {
	//Upgrade initial Get request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	//Make sure we close the connection when the function return
	defer ws.Close()

	//Register our new client
	clients[ws] = true

	//next piece is an inifinite loop that continuously waits for a new message to be
	//written to the WebSocket, unserializes it from JSON to a Message object and then
	//throws it into the broadcast channel.
	//If there is some kind of error with reading from the socket, we assume the client
	//has disconnected for some reason or another. We log the error and remove that client
	//from our global "clients" map so we don't try to read from or send new messages to that client.
	//Another thing to note is that HTTP route handler functions are run as goroutines.
	//This allows the HTTP server to handle multiple incoming connections without having
	//to wait for another connection to finish.

	for {
		var msg Message
		//Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		//Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

//Next function is simply a loop that continuously reads from the "broadcast" channel and then
//relays the message to all of our clients over their respective WebSocket connection.
func handleMessages() {
	for {
		//Grab the next message from the broadcast channel
		msg := <-broadcast
		//Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
