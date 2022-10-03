package prevotes

import (
	"fmt"
	"net/url"
)

type RPCAddress struct {
	Host string
	Port string
	TLS  bool
}

func (r *RPCAddress) HTTPRoute(path string) string {
	return fmt.Sprintf("%s://%s:%s/%s", r.HTTPScheme(), r.Host, r.Port, path)
}
func (r *RPCAddress) HTTPScheme() string {
	if r.TLS {
		return "https"
	}
	return "http"
}

func NewRPCAddress(addr string) (*RPCAddress, error) {
	parsedAddress, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	rpcAddr := RPCAddress{
		Host: parsedAddress.Hostname(),
		Port: parsedAddress.Port(),
		TLS:  false,
	}
	switch parsedAddress.Scheme {
	case "tls", "https":
		rpcAddr.TLS = true
	}
	if rpcAddr.Port == "" {
		if rpcAddr.TLS {
			rpcAddr.Port = "443"
		} else {
			rpcAddr.Port = "26657"
		}
	}

	return &rpcAddr, nil
}
