package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
)

type Request struct {
	Method string              `json:"method"`
	URL    string              `json:"url"`
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
}

type Response struct {
	StatusCode int                 `json:"statusCode"`
	Header     map[string][]string `json:"header"`
	Body       string              `json:"body"`
}

type RecordedRequest struct {
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

type Mock struct {
	Id       string   `json:"id"`
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

type State struct {
	recordedRequests []RecordedRequest
	mocks            []Mock
	mocksCount       int
}

func (s *State) saveRequest(r RecordedRequest) {
	s.recordedRequests = append(s.recordedRequests, r)
}

func buildProxyHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("Proxy-Connection")

		requestBodyByteArray, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatalf("Failed to read request body: %v", err)
		}
        requestBody := string(requestBodyByteArray)

		for _, mock := range state.mocks {
			if mock.Request.Method != r.Method {
				continue
			}
			if mock.Request.URL != r.URL.String() {
				continue
			}
			if !reflect.DeepEqual(mock.Request.Header, r.Header) {
				continue
			}
			if mock.Request.Body != string(requestBody) {
				continue
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(mock.Response.StatusCode)
			fmt.Fprintf(w, "%v", mock.Response.Body)
			return
		}

		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			state.saveRequest(RecordedRequest{
				Request: Request{
					Method: r.Method,
					URL:    r.URL.String(),
					Header: r.Header,
					Body:   string(requestBody),
				},
				Response: Response{
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
			Request: Request{
				Method: r.Method,
				URL:    r.URL.String(),
				Header: r.Header,
				Body:   string(requestBody),
			},
			Response: Response{
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

		var mock Mock
		err = json.Unmarshal(requestBody, &mock)
		if err != nil {
			panic(err)
		}

		toReturn, err := json.Marshal(mock)
		if err != nil {
			panic(err)
		}

		fmt.Printf("\n\nadded mock request body\n %s\n\n", string(toReturn))

		state.mocks = append(state.mocks, mock)
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
		toReturn, err := json.Marshal(state.recordedRequests)
		if err != nil {
			panic(err)
		}
		fmt.Println(toReturn)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%v", string(toReturn))
	}
}

func main() {
	state := &State{}

	http.HandleFunc("/addMock", buildAddMockHandler(state))
	http.HandleFunc("/mocks", buildMocksHandler(state))
	http.HandleFunc("/recordedRequests", buildRecordedRequestsHandler(state))
	http.HandleFunc("/", buildProxyHandler(state))

	fmt.Println("listening 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
