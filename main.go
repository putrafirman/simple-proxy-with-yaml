package main

import (
	"io"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration file
type Config struct {
	Routes []Route `yaml:"routes"`
}

// Route represents a single forwarding rule
type Route struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Load configuration
	config, err := loadConfig("config.yaml")
	if err != nil {
		e.Logger.Fatalf("Failed to load config: %v", err)
	}

	// Set up routes based on config
	for _, route := range config.Routes {
		setupRoute(e, route)
	}

	// Start server
	e.Logger.Fatal(e.Start(":5000"))
}

// loadConfig reads the YAML configuration file and unmarshals it into a Config struct
func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// setupRoute sets up route handlers for each HTTP method
func setupRoute(e *echo.Echo, route Route) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

	for _, method := range methods {
		route := route // capture range variable
		e.Add(method, route.From, func(c echo.Context) error {
			target := route.To + c.Path()
			e.Logger.Printf("Forwarding %s request from %s to %s\n", c.Request().Method, c.Request().RequestURI, target)
			return proxyRequest(c, target)
		})
	}
}

// proxyRequest forwards the request to the target URL
func proxyRequest(c echo.Context, target string) error {
	req, err := http.NewRequest(c.Request().Method, target, c.Request().Body)
	if err != nil {
		return err
	}

	// Copy headers from the original request
	for name, values := range c.Request().Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Copy headers from the response
	for name, values := range resp.Header {
		for _, value := range values {
			c.Response().Header().Add(name, value)
		}
	}

	c.Response().WriteHeader(resp.StatusCode)
	_, err = io.Copy(c.Response().Writer, resp.Body)
	// _, err = c.Response().Write([]byte(target))
	return err
}
