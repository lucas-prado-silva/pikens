package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type OutboundRequest struct {
	Method string
	URL    *url.URL
	Header map[string][]string
	Body   []byte
}

type OutboundRequestResponse struct {
	StatusCode int
	Header     map[string][]string
	Body       []byte
}

type RecordedRequest struct {
	outboundRequest         OutboundRequest
	outboundRequestResponse OutboundRequestResponse
}

type State struct {
	recordedRequests []RecordedRequest
	mocks            []http.Request
}

func (s *State) saveRequest(r RecordedRequest) {
	s.recordedRequests = append(s.recordedRequests, r)
}

func buildProxyHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleProxy(w, r, state)
	}
}

func handleProxy(w http.ResponseWriter, r *http.Request, state *State) {
	r.Header.Del("Proxy-Connection")

	fmt.Printf("proxy: %s %s\n", r.Method, r.URL.String())

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("Failed to read request body: %v", err)
	}
	fmt.Printf("request body %s\n", requestBody)

	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		// responseBody, err := io.ReadAll(resp.Body)
		// if err != nil {
		// 	log.Fatalf("Failed to read response body: %v", err)
		// }
		// fmt.Printf("response body %s\n", responseBody)

		state.saveRequest(RecordedRequest{
			outboundRequest: OutboundRequest{
				Method: r.Method,
				URL:    r.URL,
				Header: r.Header,
				Body:   requestBody,
			},
			outboundRequestResponse: OutboundRequestResponse{
				StatusCode: -1,
				Header:     nil,
				Body:       nil,
			},
		})
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}
	state.saveRequest(RecordedRequest{
		outboundRequest: OutboundRequest{
			Method: r.Method,
			URL:    r.URL,
			Header: r.Header,
			Body:   requestBody,
		},
		outboundRequestResponse: OutboundRequestResponse{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Body:       responseBody,
		},
	})
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func buildRecordedRequestsHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("recordedRequests")
		fmt.Println(state)
		fmt.Fprintf(w, "Recorded requests: %v", state.recordedRequests)
	}
}

func buildAddMockRequestHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Recorded requests: %v", state.recordedRequests)
	}
}

func main() {
	state := &State{}

	// http.HandleFunc("/addMock", buildRecordedRequestsHandler(state))
	http.HandleFunc("/recordedRequests", buildRecordedRequestsHandler(state))
	http.HandleFunc("/", buildProxyHandler(state))

	log.Println("Starting proxy server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
