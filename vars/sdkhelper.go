package vars

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"

	"github.com/digital-dream-labs/vector-go-sdk/pkg/vector"
	"github.com/digital-dream-labs/vector-go-sdk/pkg/vectorpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var guidLocation string = "/run/vic-cloud/perRuntimeToken"

// tokenAuth matches vector-go-sdk's per-RPC bearer auth (unexported there).
type tokenAuth struct {
	token string
}

func (t tokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (tokenAuth) RequireTransportSecurity() bool {
	return true
}

var (
	vecMu     sync.Mutex
	vecCached *vector.Vector
	vecConn   *grpc.ClientConn
	vecGUID   string
)

func GetGUID() (string, error) {
	return ReadFile(guidLocation)
}

// GetVec returns a process-wide SDK client to localhost:443.
// Previous code dialed a new gRPC/TLS connection on every call and never
// closed it (Battery UI polls every 3s) — that ballooned wired RSS to ~180MB.
func GetVec() (*vector.Vector, error) {
	guid, err := ReadFile(guidLocation)
	if err != nil {
		return nil, errors.New("empty perruntimetoken")
	}

	vecMu.Lock()
	defer vecMu.Unlock()

	if vecCached != nil && vecGUID == guid && vecConn != nil {
		return vecCached, nil
	}

	if vecConn != nil {
		_ = vecConn.Close()
		vecConn = nil
		vecCached = nil
		vecGUID = ""
	}

	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec // robot local gateway
	// vic-cloud gateway listens on IPv6 (:443 / tcp6). Dialing 127.0.0.1 fails when
	// net.ipv6.bindv6only=1; [::1] matches the live listener on Vector.
	conn, err := grpc.Dial(
		"[::1]:443",
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(tokenAuth{token: guid}),
	)
	if err != nil {
		return nil, err
	}

	v := &vector.Vector{
		Conn: vectorpb.NewExternalInterfaceClient(conn),
	}
	vecCached = v
	vecConn = conn
	vecGUID = guid
	return v, nil
}

// InvalidateVec drops the cached SDK connection so the next GetVec redials.
// Call after transport / UNAVAILABLE errors (e.g. vic-cloud restart).
func InvalidateVec() {
	vecMu.Lock()
	defer vecMu.Unlock()
	if vecConn != nil {
		_ = vecConn.Close()
		vecConn = nil
	}
	vecCached = nil
	vecGUID = ""
}
