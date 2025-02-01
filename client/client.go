package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"

	bbt "github.com/charmbracelet/bubbletea/v2"
)

type Client struct {
	base *url.URL
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}

func New(host string) Client {
	return Client{base: Must(url.Parse(host))}
}

type ErrMsg error

type Response[T any] struct {
	io.ReadCloser

	sync.Once
	*bufio.Scanner
}

func (r *Response[T]) Scan() bool {
	r.Do(func() {
		r.Scanner = bufio.NewScanner(r)
	})

	return r.Scanner.Scan()
}

func Send[T any](c Client, method, path string, requestBody any) bbt.Cmd {
	return func() bbt.Msg {
		var b bytes.Buffer
		if err := json.NewEncoder(&b).Encode(requestBody); err != nil {
			return err
		}

		request, err := http.NewRequest(method, c.base.JoinPath(path).String(), &b)
		if err != nil {
			return err
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return err
		}

		if response.StatusCode >= http.StatusBadRequest {
			bts, err := io.ReadAll(response.Body)
			if err != nil {
				return errors.New(response.Status)
			}

			return errors.New(string(bts))
		}

		return &Response[T]{
			ReadCloser: response.Body,
		}
	}
}
