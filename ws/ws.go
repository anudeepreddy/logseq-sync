package ws

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
)

type client struct {
	msg chan ([]byte)
}

type graph struct {
	clients map[*client]bool
}

type WsServer struct {
	graphs   map[string]*graph
	graphsMu sync.Mutex
}

func (s *WsServer) Handler(w http.ResponseWriter, r *http.Request) {
	graphuuid := r.URL.Query().Get("graphuuid")
	if graphuuid == "" {
		log.Printf("Failed to accept connection: Graph UUID not specified")
		return
	}

	ctx := r.Context()
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		log.Printf("Failed to accept connection: %v", err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "")

	s.joinGraph(ctx, c, graphuuid)
}

func NewWsServer() *WsServer {
	s := &WsServer{
		graphs: make(map[string]*graph),
	}
	return s
}

func (s *WsServer) createGraph(id string) *graph {
	graph := &graph{
		clients: make(map[*client]bool),
	}
	s.graphs[id] = graph
	return graph
}

func (s *WsServer) joinGraph(ctx context.Context, c *websocket.Conn, graphId string) error {

	cl := &client{
		msg: make(chan []byte),
	}
	if s.graphs[graphId] == nil {
		s.graphs[graphId] = s.createGraph(graphId)
	}
	s.graphs[graphId].clients[cl] = true
	for {
		msg := <-cl.msg
		err := writeMessage(ctx, c, msg)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
	}
}

func (s *WsServer) PublishTxnUpdate(msg []byte, graphId string) {
	s.graphsMu.Lock()
	defer s.graphsMu.Unlock()

	for c := range s.graphs[graphId].clients {
		c.msg <- msg
	}
}

func writeMessage(ctx context.Context, c *websocket.Conn, msg []byte) error {
	return c.Write(ctx, websocket.MessageText, msg)
}
