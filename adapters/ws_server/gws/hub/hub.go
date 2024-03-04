package _gws_hub

import (
	_logger "github.com/dacalin/ws_gateway/logger"
	_connection_id "github.com/dacalin/ws_gateway/models/connection_id"
	_iconnection "github.com/dacalin/ws_gateway/ports/connection"
	_ihub "github.com/dacalin/ws_gateway/ports/hub"
	"github.com/dacalin/ws_gateway/ports/pubsub"
	"sync"
)

var _ _ihub.Hub = (*Hub)(nil)

type Hub struct {
	connections sync.Map
	//connections map[_connection_id.ConnectionId]ConnectionData
	pubsub _ipubsub.Client
}

var lock = &sync.Mutex{}
var instance *Hub

func Instance() *Hub {
	return instance
}

func New(pubsub _ipubsub.Client) *Hub {
	lock.Lock()
	defer lock.Unlock()
	if instance == nil {
		instance = &Hub{
			connections: sync.Map{},
			pubsub:      pubsub,
		}
	}

	return instance
}

func listener(data ConnectionData, pubsub _ipubsub.Client) {
	_logger.Instance().Printf("listening cid=%s", data.connection.ConnectionId())

	cid := data.connection.ConnectionId()
	subscriber := pubsub.Subscribe(cid.Value())

	for {
		select {
		case signal := <-data.endSignal:
			_logger.Instance().Printf("listener end signal cid=%s", data.connection.ConnectionId())

			if signal == true {
				subscriber.Close()
				return
			}

		case msg := <-subscriber.Receive():
			_logger.Instance().Printf("RECEIVER MSG cid=%s, MSG=%s", data.connection.ConnectionId(), msg)
			data.connection.Send(msg)

		}
	}

}

func (self *Hub) Set(cid _connection_id.ConnectionId, conn _iconnection.Connection) {

	channel := make(chan bool)

	data := ConnectionData{
		endSignal:  channel,
		connection: conn,
	}

	self.connections.Store(cid, data)

	go listener(data, self.pubsub)
}

func (self *Hub) Get(cid _connection_id.ConnectionId) (_iconnection.Connection, bool) {
	conn, found := self.connections.Load(cid)

	if found == false {
		return nil, found
	}

	connCasted := conn.(ConnectionData)
	return connCasted.connection, found
}

func (self *Hub) Delete(cid _connection_id.ConnectionId) {
	conn, found := self.connections.Load(cid)
	if found == false {
		return
	}

	connCasted := conn.(ConnectionData)
	connCasted.endSignal <- true

	self.connections.Delete(cid)
}

func (self *Hub) PubSub() _ipubsub.Client {
	return self.pubsub
}

func (self *Hub) Send(cid _connection_id.ConnectionId, data []byte) {
	_logger.Instance().Printf("Send To cid=%s", cid.Value())

	conn, found := self.Get(cid)

	if found == false {
		_logger.Instance().Println("Send using PubSub")
		self.PubSub().Publish(cid.Value(), data)
	} else {
		conn.Send(data)

	}
}
