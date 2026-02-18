package middleware

import "net/http"

type Chain struct {
	middlewares []func(http.Handler) http.Handler
}

func NewChain() *Chain {
	return &Chain{
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}

func (c *Chain) Use(middleware func(http.Handler) http.Handler) *Chain {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

func (c *Chain) Handle(handler http.Handler) http.Handler {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}
