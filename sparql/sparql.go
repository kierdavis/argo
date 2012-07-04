package sparql

import (
	"fmt"
	"io"
	"net/http"
)

func EnsureOK(resp *http.Response, err error) (respR *http.Response, errR error) {
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		_, err = DropBody(resp, nil)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("HTTP request returned %s", resp.Status)
	}

	return resp, nil
}

func DropBody(resp *http.Response, err error) (respR *http.Response, errR error) {
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, 1024)

	for {
		_, err := resp.Body.Read(buffer)
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	return resp, nil
}
