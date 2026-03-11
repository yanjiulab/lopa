package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/yanjiulab/lopa/internal/config"
	"github.com/yanjiulab/lopa/internal/logger"
	"github.com/yanjiulab/lopa/internal/measurement"
)

// Start starts the HTTP server in a separate goroutine.
func Start() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	registerRoutes(e)

	go func() {
		addr := config.Global().HTTP.Addr
		if addr == "" {
			addr = ":8080"
		}
		logger.S().Infow("starting http server", "addr", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.S().Errorw("http server error", "err", err)
		}
	}()

	return e
}

func registerRoutes(e *echo.Echo) {
	g := e.Group("/api/v1")

	g.POST("/tasks/ping", createPingTaskHandler)
	g.POST("/tasks/tcp", createTcpTaskHandler)
	g.POST("/tasks/udp", createUdpTaskHandler)
	g.POST("/tasks/twamp", createTwampTaskHandler)
	g.GET("/tasks", listTasksHandler)
	g.GET("/tasks/:id", getTaskHandler)
	g.POST("/tasks/:id/stop", stopTaskHandler)
	g.DELETE("/tasks/:id", deleteTaskHandler)
}

type createPingRequest struct {
	measurement.TaskParams `json:",inline"`
}

type createTaskResponse struct {
	ID string `json:"id"`
}

func createPingTaskHandler(c echo.Context) error {
	var req measurement.TaskParams
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	req.Type = "ping"

	engine := measurement.DefaultEngine()
	id, err := engine.CreatePingTask(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, createTaskResponse{ID: string(id)})
}

func createTcpTaskHandler(c echo.Context) error {
	var req measurement.TaskParams
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	req.Type = "tcp"

	engine := measurement.DefaultEngine()
	id, err := engine.CreateTcpTask(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, createTaskResponse{ID: string(id)})
}

func createUdpTaskHandler(c echo.Context) error {
	var req measurement.TaskParams
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	req.Type = "udp"

	engine := measurement.DefaultEngine()
	id, err := engine.CreateUdpTask(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, createTaskResponse{ID: string(id)})
}

func createTwampTaskHandler(c echo.Context) error {
	var req measurement.TaskParams
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	req.Type = "twamp"

	engine := measurement.DefaultEngine()
	id, err := engine.CreateTwampTask(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, createTaskResponse{ID: string(id)})
}

func listTasksHandler(c echo.Context) error {
	engine := measurement.DefaultEngine()
	results := engine.ListResults()
	return c.JSON(http.StatusOK, results)
}

func getTaskHandler(c echo.Context) error {
	id := measurement.TaskID(c.Param("id"))
	engine := measurement.DefaultEngine()
	res, ok := engine.GetResult(id)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}
	return c.JSON(http.StatusOK, res)
}

func stopTaskHandler(c echo.Context) error {
	id := measurement.TaskID(c.Param("id"))
	engine := measurement.DefaultEngine()
	engine.StopTask(id)
	return c.NoContent(http.StatusAccepted)
}

func deleteTaskHandler(c echo.Context) error {
	id := measurement.TaskID(c.Param("id"))
	engine := measurement.DefaultEngine()
	ok := engine.DeleteTask(id)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}
	return c.NoContent(http.StatusNoContent)
}

