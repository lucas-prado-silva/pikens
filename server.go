package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type OutboundRequest struct {
	Method string              `json:"method"`
	URL    string              `json:"url"`
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

type OutboundRequestResponse struct {
	StatusCode int                 `json:"statusCode"`
	Header     map[string][]string `json:"header"`
	Body       string              `json:"body"`
}

type RecordedRequest struct {
	OutboundRequest         OutboundRequest         `json:"outboundRequest"`
	OutboundRequestResponse OutboundRequestResponse `json:"outboundRequestResponse"`
}

type State struct {
	recordedRequests []RecordedRequest
	mocks            []RecordedRequest
}

func (s *State) saveRequest(r RecordedRequest) {
	s.recordedRequests = append(s.recordedRequests, r)
}

func buildProxyHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("Proxy-Connection")

		fmt.Printf("proxy: %s %s\n", r.Method, r.URL.String())

		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatalf("Failed to read request body: %v", err)
		}
		fmt.Printf("request body %s\n", requestBody)

		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			state.saveRequest(RecordedRequest{
				OutboundRequest: OutboundRequest{
					Method: r.Method,
					URL:    r.URL.String(),
					Header: r.Header,
					Body:   string(requestBody),
				},
				OutboundRequestResponse: OutboundRequestResponse{
					StatusCode: -1,
					Header:     nil,
					Body:       "",
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
			OutboundRequest: OutboundRequest{
				Method: r.Method,
				URL:    r.URL.String(),
				Header: r.Header,
				Body:   string(requestBody),
			},
			OutboundRequestResponse: OutboundRequestResponse{
				StatusCode: resp.StatusCode,
				Header:     resp.Header,
				Body:       string(responseBody),
			},
		})
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func buildAddMockHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatalf("Failed to read request body: %v", err)
		}
		fmt.Printf("\n\nadding mock request body\n %s\n\n", requestBody)

		var recordedRequest RecordedRequest
		err = json.Unmarshal(requestBody, &recordedRequest)
		if err != nil {
			panic(err)
		}

		state.mocks = append(state.mocks, recordedRequest)

        toReturn, err := json.Marshal(recordedRequest)
        if err != nil {
            panic(err)
        }

		fmt.Printf("\n\nadded mock request body\n %s\n\n", string(toReturn))
		fmt.Fprintf(w, "%v", string(toReturn))
	}
}

func buildMocksHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
        toReturn, err := json.Marshal(state.mocks)
        if err != nil {
            panic(err)
        }
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%v", string(toReturn))
	}
}

func buildRecordedRequestsHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("recordedRequests")
		fmt.Println(state)
		fmt.Fprintf(w, "Current recorded requests in state: %v", state.recordedRequests)
	}
}

func main() {
	state := &State{}

	http.HandleFunc("/addMock", buildAddMockHandler(state))
	http.HandleFunc("/mocks", buildMocksHandler(state))
	http.HandleFunc("/recordedRequests", buildRecordedRequestsHandler(state))
	http.HandleFunc("/", buildProxyHandler(state))

	log.Println("Starting proxy server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
