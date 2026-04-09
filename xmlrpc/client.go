package xmlrpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/rpc"
	"net/url"
	"sync"
)

// maxResponseBytes is the maximum number of bytes read from an XML-RPC response
// body. It prevents a misbehaving or malicious server from exhausting memory.
const maxResponseBytes = 32 << 20 // 32 MiB

// Client wraps rpc.Client with context-aware call support.
// callMu serialises concurrent CallContext invocations so that the shared
// codec context field is never overwritten by a racing goroutine.
type Client struct {
	*rpc.Client
	codec  *clientCodec
	callMu sync.Mutex
}

// clientCodec is rpc.ClientCodec interface implementation.
type clientCodec struct {
	// url presents url of xmlrpc service
	url *url.URL

	// httpClient works with HTTP protocol
	httpClient *http.Client

	// cookies stores cookies received on last request
	cookies http.CookieJar

	// responses presents map of active requests. It is required to return request id, that
	// rpc.Client can mark them as done.
	responses map[uint64]*http.Response
	mutex     sync.Mutex

	response Response

	// ctx is the context for the current in-flight request. Access is
	// serialised by Client.callMu — no additional locking is needed here.
	ctx context.Context

	// ready presents channel, that is used to link request and its response.
	ready chan uint64

	// close notifies codec is closed.
	close chan struct{}
}

func (codec *clientCodec) WriteRequest(request *rpc.Request, args interface{}) (err error) {
	httpRequest, err := NewRequest(codec.ctx, codec.url.String(), request.ServiceMethod, args)
	if err != nil {
		return err
	}

	if codec.cookies != nil {
		for _, cookie := range codec.cookies.Cookies(codec.url) {
			httpRequest.AddCookie(cookie)
		}
	}

	var httpResponse *http.Response
	httpResponse, err = codec.httpClient.Do(httpRequest)
	if err != nil {
		return err
	}

	if codec.cookies != nil {
		codec.cookies.SetCookies(codec.url, httpResponse.Cookies())
	}

	codec.mutex.Lock()
	codec.responses[request.Seq] = httpResponse
	codec.mutex.Unlock()

	codec.ready <- request.Seq

	return nil
}

func (codec *clientCodec) ReadResponseHeader(response *rpc.Response) (err error) {
	var seq uint64
	select {
	case seq = <-codec.ready:
	case <-codec.close:
		return errors.New("codec is closed")
	}
	response.Seq = seq

	codec.mutex.Lock()
	httpResponse := codec.responses[seq]
	delete(codec.responses, seq)
	codec.mutex.Unlock()

	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		response.Error = fmt.Sprintf("request error: bad status code - %d", httpResponse.StatusCode)
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxResponseBytes))
	if err != nil {
		response.Error = err.Error()
		return nil
	}

	resp := Response(body)
	if err := resp.Err(); err != nil {
		response.Error = err.Error()
		return nil
	}

	codec.response = resp

	return nil
}

func (codec *clientCodec) ReadResponseBody(v interface{}) (err error) {
	if v == nil {
		return nil
	}
	return codec.response.Unmarshal(v)
}

func (codec *clientCodec) Close() error {
	if transport, ok := codec.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	close(codec.close)

	return nil
}

// Call invokes the named function, waits for it to complete, and returns its
// error status. It is context-aware: cancellation and deadlines are propagated
// to the underlying HTTP request.
func (c *Client) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return c.CallContext(context.Background(), serviceMethod, args, reply)
}

// CallContext is like Call but honours the supplied context for cancellation
// and deadline propagation into the underlying HTTP request.
//
// callMu serialises concurrent calls so that the context stored on the codec
// is never overwritten by a racing goroutine before WriteRequest reads it.
func (c *Client) CallContext(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	c.callMu.Lock()
	c.codec.ctx = ctx
	err := c.Client.Call(serviceMethod, args, reply)
	c.callMu.Unlock()
	return err
}

// NewClient returns instance of rpc.Client object, that is used to send request to xmlrpc service.
func NewClient(requrl string, transport http.RoundTripper) (*Client, error) {
	if transport == nil {
		transport = http.DefaultTransport
	}

	httpClient := &http.Client{Transport: transport}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(requrl)
	if err != nil {
		return nil, err
	}

	codec := &clientCodec{
		url:        u,
		httpClient: httpClient,
		close:      make(chan struct{}),
		ready:      make(chan uint64),
		responses:  make(map[uint64]*http.Response),
		cookies:    jar,
		ctx:        context.Background(),
	}

	return &Client{rpc.NewClientWithCodec(codec), codec, sync.Mutex{}}, nil
}
