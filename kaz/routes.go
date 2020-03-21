package kaz

import (
	"fmt"
	"net/http"

	"github.com/flosch/pongo2"
	"github.com/gin-gonic/gin"
)

func routes(s *Server) error {
	r := s.Engine
	l := s.Logger
	if r == nil {
		return fmt.Errorf("unable to initialize routes: Gin engine is nil")
	}

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", pongo2.Context{
			"title": "TEAMS",
		})
	})

	r.GET("/cache", func(c *gin.Context) {
		err := InitializeCache(true)
		c.HTML(200, "success.html", pongo2.Context{
			"title": "CACHE",
			"error": err,
		})
	})

	t := r.Group("/clients")
	{
		t.POST("/checkin", func(c *gin.Context) {
			var x struct {
				MacAddress string `json:"MacAddress"`
				Os         string `json:"Os"`
				OsVersion  string `json:"OsVersion"`
			}
			err := c.BindJSON(&x)

			if err != nil {
				l.Printf("Error Binding Params: %+v", err)
				c.JSON(http.StatusBadRequest, err)
			}
			l.Printf("Bound Params: %+v", x)

			var w Client
			s.Db.Where(&Client{MacAddress: x.MacAddress}).First(&w)

			if w.CheckedIn {
				c.JSON(200, w)
				return
			}

			h, err := GetClientByMacAddress(x.MacAddress)
			if err != nil {
				c.String(500, "Unable to complete request: %+v", err)
				return
			}

			h.CheckedIn = true

			err = CommitClient(h, s)
			if err != nil {
				c.String(500, "Unable to complete request: %+v", err)
				return
			}

			c.JSON(200, h)
		})
	}

	return nil
}
