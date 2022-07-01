package main

import (
	"bytes"
	"fmt"
	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"io"
	"io/ioutil"
	"net/http"
)

func main() {
	e := echo.New()
	e.Debug = true
	e.HideBanner = true

	e.Use(middleware.Logger())
	c := jaegertracing.New(e, nil)
	defer c.Close()

	// This is only for example
	client := http.DefaultClient

	e.GET("/broken", func(c echo.Context) error {
		span := jaegertracing.CreateChildSpan(c, "Test broken body")
		defer span.Finish()

		body := bytes.NewBufferString("Hello jaeger!")

		tracedRequest, _ := jaegertracing.NewTracedRequest(
			"POST",
			"http://0.0.0.0:1337/body",
			body,
			span,
		)

		response, err := client.Do(tracedRequest)
		defer response.Body.Close()

		if err != nil {
			return c.String(
				http.StatusInternalServerError,
				err.Error(),
			)
		}

		received, _ := ioutil.ReadAll(response.Body)
		receivedStr := fmt.Sprintf("Received: %s", string(received))

		return c.String(
			response.StatusCode,
			receivedStr,
		)
	})
	e.GET("/fixed", func(c echo.Context) error {
		span := jaegertracing.CreateChildSpan(c, "Test fixed body")
		defer span.Finish()

		body := bytes.NewBufferString("Hello jaeger!")

		tracedRequest, _ := NewPatchedTracedRequest(
			"POST",
			"http://0.0.0.0:1337/body",
			body,
			span,
		)

		response, err := client.Do(tracedRequest)
		defer response.Body.Close()

		if err != nil {
			return c.String(
				http.StatusInternalServerError,
				"Internal request failed",
			)
		}

		received, _ := ioutil.ReadAll(response.Body)
		receivedStr := fmt.Sprintf("Received: %s", string(received))

		return c.String(
			response.StatusCode,
			receivedStr,
		)
	})
	e.POST("/body", func(c echo.Context) error {
		body, _ := ioutil.ReadAll(c.Request().Body)

		if len(body) == 0 {
			return c.String(
				http.StatusBadRequest,
				"Body is missing",
			)
		}

		return c.String(
			http.StatusOK,
			fmt.Sprintf("Body is OK: %s", string(body)),
		)
	})
	e.Logger.Fatal(e.Start(":1337"))
}

// See https://github.com/labstack/echo-contrib/pull/71
func NewPatchedTracedRequest(
	method string,
	url string,
	body io.Reader,
	span opentracing.Span,
) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err.Error())
	}

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, method)
	span.Tracer().Inject(span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header))

	return req, err
}
