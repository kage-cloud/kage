package controller

import "github.com/labstack/echo/v4"

type Controller interface {
	Routes() []Route
	Group() string
}

type Route struct {
	Path    string
	Method  string
	Handler echo.HandlerFunc
}
