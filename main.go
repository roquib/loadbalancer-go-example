package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, req http.Request)
}

type SimpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func newSimpleServer(addr string) *SimpleServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	return &SimpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}

}

type LoadBalancer struct {
	port            string
	roundrobinCount int
	servers         []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundrobinCount: 0,
		servers:         servers,
	}
}

func (s *SimpleServer) Address() string { return s.addr }

func (s *SimpleServer) IsAlive() bool { return true }
func (s *SimpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

func (lb *LoadBalancer) getNexAvailableServer() Server {
	server := lb.servers[lb.roundrobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundrobinCount++
		server = lb.servers[lb.roundrobinCount%len(lb.servers)]
	}
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNexAvailableServer()
	fmt.Printf("forwaring request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, req)
}
func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.bing.com"),
		newSimpleServer("https://www.duckduckgo.com"),
	}

	lb := NewLoadBalancer("8000", servers)

	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}

	http.HandlerFunc("/", handleRedirect)

	fmt.Printf("serving requests at `localhost:%s`\n", lb.port)
	log.Fatal(http.ListenAndServe(":"+lb.port, nil))
}
