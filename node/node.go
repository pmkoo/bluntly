package node

import (
  "github.com/nictuku/dht"
  "net"
  "crypto/rsa"
  "crypto/x509"
  "sync"
  "fmt"
  "io"
  "encoding/binary"
  "errors"
  "golang.org/x/crypto/nacl/box"
  "crypto/rand"
)

/* NODE */
type Node struct { 
  dht *dht.DHT
  config *Config
}

/* CONFIG */
type FileConfig struct {
  dhtPort int
  ownKeyFile string
  id string
  configRoot string
  holePunch HolePunchConf
}

type Config struct {
  dhtPort int
  ownKey rsa.PrivateKey
  id string
  configRoot string
  holePunch HolePunchConf
  contactList *ContactList
}

type HolePunchConf struct {
	recvPort int
}

type ContactList struct {
  contacts *map[rsa.PublicKey]string
  mut *sync.Mutex
}

func NewNode(conf *Config) (node *Node, err error) {
  // setup the DHT
  dhtConf := dht.DefaultConfig
  dhtConf.Port = conf.dhtPort
  dht, err := dht.New(dhtConf)
  if err != nil { return }
  go dht.Run()

  node.dht = dht

	return node, nil
}

/* CLIENT */

/* LISTENER */

type Listener struct {
  connChan chan *Conn
}

func (n *Node) Listen(port int) (listener *Listener, err error) {
  connChan := make(chan *Conn)
  listener = &Listener{connChan: connChan}
  // setup TCP listener
  tcpListener, err := net.Listen("tcp", fmt.Sprintf(":"))
  if err != nil {return}

  // loop accepting TCP connections
  go func() {
    for {
      tcpConn, tcpErr := tcpListener.Accept()
      if tcpErr != nil {
       Log(LOG_ERROR, "%s", tcpErr)
      }

      go func() {
        conn, handshakeError := handleClientConn(tcpConn, &n.config.ownKey, n.config.contactList)
        if err != nil {
          Log(LOG_INFO,
              "handling client connection from address %s %s",
              tcpConn.RemoteAddr().String(), handshakeError)
        } else {
          connChan <- conn
        }
      }()    
    }
  }()

  return
}

func (l *Listener) Accept() (c net.Conn, err error) {
  conn := <- l.connChan
  return conn, nil
}

func (l *Listener) Close() error {
  return nil
}

func Addr() net.Addr {
  return nil
}

func handleClientConn(rawConn net.Conn,
                      ownKey *rsa.PrivateKey,
                      contacts *ContactList) (conn *Conn, err error) {

  var handshakeLen int64
  err = binary.Read(rawConn, binary.BigEndian, &handshakeLen)
  if (err != nil) { return }

  ciphertext := make([]byte, handshakeLen)
  _, err = io.ReadFull(rawConn, ciphertext)
  if (err != nil) { return }

  plain, err := rsa.DecryptPKCS1v15(nil, ownKey, ciphertext)
  if (err != nil) { return }

  connReq := ConnRequest{}
  err = connReq.UnmarshalBinary(plain)
  if (err != nil) { return }

  _, privKey, err := box.GenerateKey(rand.Reader)
  if (err != nil) { return }

  var sharedKey [32]byte
  box.Precompute(&sharedKey, connReq.sessionKey, privKey) 

  return &Conn{rawConn, &sharedKey}, nil
}


const sessionKeyLen = 32

/* connection request */
type ConnRequest struct {
  peerPub *rsa.PublicKey
  sessionKey *[32]byte 
}

type ConnResponse struct {

}

func (r *ConnRequest) MarshalBinary() (data []byte, err error) {
  pubKeyBytes, err := x509.MarshalPKIXPublicKey(r.peerPub)
  if (err != nil) { return }

  return append((*r.sessionKey)[:], pubKeyBytes...), nil
}

func (r *ConnRequest) UnmarshalBinary(data []byte) (err error) {
  copiedBytes := copy(r.sessionKey[:], data[:32])
  if (copiedBytes < 32) {
    return errors.New("session key too short.")    
  }

  someKey, err := x509.ParsePKIXPublicKey(data)
  pubKey := someKey.(*rsa.PublicKey)
  if (err != nil) { return }
  r.peerPub = pubKey

  return
}

/* CONNECTION */


type Conn struct {
  net.Conn // underlying network connection 
  sharedKey *[32]byte
}

func (c *Conn) Read(b []byte) (n int, err error) {
  return
}

func (c *Conn) Write(b []byte) (n int, err error) {
  return 

}
func (c Conn) Close() error {
  return nil
}



