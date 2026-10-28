// Copyright 2026 The Caddy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package caddyhttp

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestHTTPRedirectListenerRequestTargets(t *testing.T) {
	for _, tc := range []struct {
		name, request, location string
	}{
		{
			name:     "origin form",
			request:  "GET /x?q=1 HTTP/1.1\r\nHost: test.localhost:8446\r\n\r\n",
			location: "https://test.localhost:8446/x?q=1",
		},
		{
			name:     "absolute form HEAD",
			request:  "HEAD http://evil.example/x?q=1 HTTP/1.1\r\nHost: evil.example\r\n\r\n",
			location: "https://evil.example/x?q=1",
		},
		{
			name:     "absolute form GET",
			request:  "GET http://evil.example/x HTTP/1.1\r\nHost: evil.example\r\n\r\n",
			location: "https://evil.example/x",
		},
		{
			name:     "absolute form POST",
			request:  "POST http://evil.example/x HTTP/1.1\r\nHost: evil.example\r\nContent-Length: 0\r\n\r\n",
			location: "https://evil.example/x",
		},
		{
			name:     "asterisk form",
			request:  "OPTIONS * HTTP/1.1\r\nHost: test.localhost:8446\r\n\r\n",
			location: "https://test.localhost:8446/",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, client := net.Pipe()
			defer server.Close()
			defer client.Close()
			if err := client.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			go func() {
				defer close(done)
				conn := &httpRedirectConn{Conn: server, limit: 1024 * 1024, r: bufio.NewReader(server)}
				_, _ = conn.Read(make([]byte, 4096))
			}()
			if _, err := fmt.Fprint(client, tc.request); err != nil {
				t.Fatal(err)
			}
			resp, err := http.ReadResponse(bufio.NewReader(client), &http.Request{Method: strings.Fields(tc.request)[0]})
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusPermanentRedirect {
				t.Errorf("status = %d, want 308", resp.StatusCode)
			}
			if got := resp.Header.Get("Location"); got != tc.location {
				t.Errorf("Location = %q, want %q", got, tc.location)
			}
			<-done
		})
	}
}

func TestFirstBytesLookLikeHTTPAbsoluteFormGET(t *testing.T) {
	if !firstBytesLookLikeHTTP([]byte("GET h")) {
		t.Fatal("absolute-form GET request was not recognized as HTTP")
	}
}
