package dht

import (
	"container/list"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/movsb/torrent/pkg/common"
	"github.com/zeebo/bencode"
)

// Client ...
type Client struct {
	MyNodeID NodeID
	Address  string

	conn    *net.UDPConn
	recvBuf []byte

	mu      sync.Mutex
	pending *list.List
}

type _Pending struct {
	tx          _TransactionID
	timeEnqueue time.Time
	addr        *net.UDPAddr
	m           *Message
	done        chan struct{}
}

// ListenAndServe ...
func (c *Client) ListenAndServe() error {
	c.recvBuf = make([]byte, 64<<10)
	c.pending = list.New()
	go c.tidyQueue()

	dstAddr, err := net.ResolveUDPAddr("udp", c.Address)
	if err != nil {
		return fmt.Errorf("resolve udp address failed: %v", err)
	}
	c.conn, err = net.ListenUDP(`udp`, dstAddr)
	if err != nil {
		return fmt.Errorf("listen udp address failed: %v", err)
	}
	defer c.conn.Close()
	log.Printf("listen udp address: %s", dstAddr.String())

	for {
		addr, msg, err := c.recv()
		if err != nil {
			return err
		}
		if msg.Type == 'q' {
			c.onQuery(addr, msg)
			continue
		}
		pending := c.dequeue(msg.TransactionID)
		if pending == nil {
			log.Printf("tx id %v not found", msg.TransactionID)
			continue
		}
		pending.addr = addr
		pending.m = msg
		close(pending.done)
	}
}

func (c *Client) onQuery(addr *net.UDPAddr, msg *Message) {
	log.Printf("onQuery: addr: %v, msg: %v", addr, msg)
}

func (c *Client) enqueue(tx _TransactionID) *_Pending {
	c.mu.Lock()
	defer c.mu.Unlock()
	pending := &_Pending{
		timeEnqueue: time.Now(),
		tx:          tx,
		done:        make(chan struct{}),
	}
	c.pending.PushBack(pending)
	return pending
}

func (c *Client) dequeue(tx _TransactionID) *_Pending {
	c.mu.Lock()
	defer c.mu.Unlock()
	var pending *_Pending
	for head := c.pending.Front(); head != nil; head = head.Next() {
		p := head.Value.(*_Pending)
		if p.tx == tx {
			pending = p
			c.pending.Remove(head)
			break
		}
	}
	return pending
}

func (c *Client) tidyQueue() {
	tick := time.NewTicker(time.Second * 3)
	for range tick.C {
		c.mu.Lock()
		log.Printf("enter tidy: %d\n", c.pending.Len())
		var next *list.Element
		for e := c.pending.Front(); e != nil; e = next {
			next = e.Next()
			p := e.Value.(*_Pending)
			if time.Since(p.timeEnqueue) > time.Second*5 {
				close(p.done)
				c.pending.Remove(e)
				log.Printf("tidy %v\n", p.tx)
			}
		}
		c.mu.Unlock()
	}
}

func (c *Client) sendQuery(addr string, query string, args map[string]interface{}) (*_Pending, error) {
	q := Message{
		TransactionID: makeTransactionID(),
		Type:          'q',
		Query:         query,
		Args: map[string]interface{}{
			`id`: c.MyNodeID,
		},
	}
	for k, v := range args {
		q.Args[k] = v // will override existing args
	}

	udpAddr, err := net.ResolveUDPAddr(`udp`, addr)
	if err != nil {
		return nil, fmt.Errorf("dht: invalid address: %v", err)
	}

	pending := c.enqueue(q.TransactionID)
	if err := c.send(udpAddr, &q); err != nil {
		c.dequeue(q.TransactionID)
		close(pending.done)
		return nil, err
	}

	return pending, nil
}

func (c *Client) sendResponse(addr *net.UDPAddr, tx _TransactionID, values map[string]interface{}) error {
	r := Message{
		TransactionID: tx,
		Type:          'r',
		Values: map[string]interface{}{
			`id`: c.MyNodeID,
		},
	}
	for k, v := range values {
		r.Values[k] = v // will override existing args
	}
	return c.send(addr, &r)
}

func (c *Client) sendError(addr *net.UDPAddr, tx _TransactionID, code int, message string) error {
	e := Message{
		TransactionID: tx,
		Type:          'e',
		Err: &_E{
			Code:    code,
			Message: message,
		},
	}
	return c.send(addr, &e)
}

func (c *Client) send(addr *net.UDPAddr, m *Message) error {
	b, err := bencode.EncodeBytes(m)
	fmt.Println(`send:`, string(b))
	if err != nil {
		return err
	}

	if _, err := c.conn.WriteToUDP(b, addr); err != nil {
		log.Printf("dht: failed to write udp to %s: %v", addr.String(), err)
		return err
	}
	return nil
}

func (c *Client) recv() (addr *net.UDPAddr, msg *Message, err error) {
	// defer c.conn.SetReadDeadline(time.Time{})
	// c.conn.SetReadDeadline(time.Now().Add(time.Second * 15))
	n, addr, err := c.conn.ReadFromUDP(c.recvBuf)
	if err != nil {
		err = fmt.Errorf("dht: recv udp failed: %v", err)
		return
	}
	msg = &Message{}
	if err = bencode.DecodeBytes(c.recvBuf[:n], msg); err != nil {
		err = fmt.Errorf("dht: decode recvBuf failed: %v", err)
		return
	}
	if msg.Type == 'e' {
		return
	}
	id, ok := msg.Values[`id`].(string)
	if !ok {
		err = fmt.Errorf(`dht: response has no id`)
		return
	}
	if len(id) != 20 {
		err = fmt.Errorf("dht: response id is not 20 bytes long")
		return
	}
	return
}

// Ping ...
func (c *Client) Ping(addr string) error {
	pending, err := c.sendQuery(addr, `ping`, nil)
	if err != nil {
		return fmt.Errorf("dht: ping failed: %v", err)
	}
	<-pending.done
	if pending.m == nil {
		return fmt.Errorf("dht: no message for ping")
	}
	fmt.Printf("Ping response: %v\n", pending.m)
	return nil
}

// FindNode ...
func (c *Client) FindNode(addr string, target NodeID) ([]CompactNodeInfo, error) {
	args := map[string]interface{}{
		`target`: string(target[:]),
	}
	pending, err := c.sendQuery(addr, `find_node`, args)
	if err != nil {
		return nil, err
	}
	<-pending.done
	if pending.m == nil {
		return nil, fmt.Errorf("dht: no message for find_node")
	}
	nodeValue, ok := pending.m.Values[`nodes`]
	if !ok {
		return nil, nil // empty result isn't an error, the spec says 'should'
	}
	nodesString, ok := nodeValue.(string)
	if !ok {
		return nil, fmt.Errorf("dht: response nodes is not a string")
	}
	if len(nodesString)%26 != 0 {
		return nil, fmt.Errorf("dht: response nodes is not a multiply of 26 bytes")
	}

	nodeBytes := []byte(nodesString)
	nodes := make([]CompactNodeInfo, 0, len(nodeBytes)/26)

	for i := 0; i < len(nodeBytes); i += 26 {
		var node CompactNodeInfo
		if err := node.Unmarshal(nodeBytes[i : i+26]); err != nil {
			return nil, fmt.Errorf("dht: find_node: %v", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetPeers ...
func (c *Client) GetPeers(addr string, infoHash common.Hash) (token string, peers []CompactPeerInfo, nodes []CompactNodeInfo, rErr error) {
	args := map[string]interface{}{
		`info_hash`: infoHash,
	}
	pending, err := c.sendQuery(addr, `get_peers`, args)
	if err != nil {
		rErr = err
		return
	}
	<-pending.done
	if pending.m == nil {
		rErr = fmt.Errorf("dht: no message for get_peers")
		return
	}
	r := pending.m
	tokenString, ok := r.Values[`token`].(string)
	if !ok {
		rErr = fmt.Errorf("dht: get_peers: no valid token returned")
		return
	}
	token = fmt.Sprintf("%x", tokenString)

	peerStrings, hasValues := r.Values[`values`].([]string)
	if hasValues {
		for _, value := range peerStrings {
			var peer CompactPeerInfo
			if err := peer.Unmarshal([]byte(value)); err != nil {
				rErr = fmt.Errorf("dht: get_peers: %v", err)
				return
			}
			peers = append(peers, peer)
		}
	}

	nodesString, hasNodes := r.Values[`nodes`].(string)
	if hasNodes {
		nodeBytes := []byte(nodesString)
		for i := 0; i < len(nodeBytes); i += 26 {
			var node CompactNodeInfo
			if err := node.Unmarshal(nodeBytes[i : i+26]); err != nil {
				rErr = fmt.Errorf("dht: find_node: %v", err)
				return
			}
			nodes = append(nodes, node)
		}
	}

	if (hasValues && hasNodes) || (!hasValues && !hasNodes) {
		rErr = fmt.Errorf("dht: get_peers: values and/or nodes are missing or exist both")
		return
	}

	return
}

// AnnouncePeer ...
func (c *Client) AnnouncePeer(addr string, infoHash common.Hash, port uint16, token string) error {
	args := map[string]interface{}{
		`info_hash`: infoHash,
		`port`:      port,
		`token`:     token,
	}
	pending, err := c.sendQuery(addr, `announce_peer`, args)
	if err != nil {
		return err
	}
	<-pending.done
	if pending.m == nil {
		return fmt.Errorf("dht: no message for announce_peer")
	}

	return nil
}
