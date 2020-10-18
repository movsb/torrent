package trackertcpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	trackertcpcommon "github.com/movsb/torrent/pkg/tracker/tcp/common"
	"github.com/zeebo/bencode"
)

// Server ...
type Server struct {
	endpoint string
	cache    *_PeerCache
}

// NewServer ...
func NewServer(endpoint string) *Server {
	return &Server{
		endpoint: endpoint,
		cache:    _NewPeerCache(),
	}
}

// Run ...
func (s *Server) Run(ctx context.Context) error {
	endpoint := s.endpoint
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(filepath.Join(`/`, u.Path), s.handleAnnounce)

	hs := http.Server{
		Addr:    u.Host,
		Handler: mux,
	}

	ch := make(chan error)
	go func() {
		err := hs.ListenAndServe()
		if err != http.ErrServerClosed {
			if ch != nil { // race?
				ch <- err
			}
		}
	}()
	select {
	case err := <-ch:
		close(ch)
		return err
	case <-time.After(time.Second):
		c := ch
		ch = nil // race?
		close(c)
		go func() {
			<-ctx.Done()
			hs.Shutdown(context.Background())
		}()
		return nil
	}
}

func (s *Server) handleAnnounce(w http.ResponseWriter, r *http.Request) {
	announceError := func(w http.ResponseWriter, err error) {
		w.WriteHeader(400)
		bencode.NewEncoder(w).Encode(
			&trackertcpcommon.AnnounceResponse{
				FailureReason: err.Error(),
			},
		)
	}

	var (
		infoHash [20]byte
		peerID   [20]byte
		ip       string
		port     int
	)

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		announceError(w, err)
		return
	}
	ipv4 := net.ParseIP(host).To4()
	if ipv4 == nil {
		announceError(w, fmt.Errorf("invalid remote address"))
		return
	}
	ip = ipv4.String()

	paramFuncs := map[string]func(value string) error{
		`info_hash`: func(value string) error {
			if len(value) != 20 {
				return fmt.Errorf("invalid info_hash")
			}
			copy(infoHash[:], value)
			return nil
		},
		`peer_id`: func(value string) error {
			if len(value) != 20 {
				return fmt.Errorf("invalid peer_id")
			}
			copy(peerID[:], value)
			return nil
		},
		`port`: func(value string) error {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > 65535 {
				return fmt.Errorf("invalid port")
			}
			port = n
			return nil
		},
	}

	for name, fn := range paramFuncs {
		if err := extractQuery(r, name, fn); err != nil {
			announceError(w, err)
			return
		}
	}

	peersCache := s.cache.Add(infoHash, peerID, ip, port)
	peers := []trackertcpcommon.Peer{}
	for _, c := range peersCache {
		peers = append(peers, trackertcpcommon.Peer{
			ID:   c.PeerID,
			IP:   c.IP,
			Port: c.Port,
		})
	}

	bencode.NewEncoder(w).Encode(
		&trackertcpcommon.AnnounceResponse{
			Interval: 60,
			Peers:    peers,
		},
	)
}

func extractQuery(r *http.Request, name string, converter func(value string) error) error {
	v := r.URL.Query()
	q, ok := v[name]
	if !ok {
		return fmt.Errorf("missing %s", name)
	}
	err := converter(q[0])
	if err != nil {
		return fmt.Errorf("param %s: %v", name, err)
	}
	return nil
}
