package kaz

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func routes(s *Server) error {
	r := s.Engine
	l := s.Logger
	if r == nil {
		return fmt.Errorf("unable to initialize routes: Gin engine is nil")
	}

	r.GET("/", func(c *gin.Context) {
		c.String(200, "Hallo!")
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
			}
			l.Printf("Bound Params: %+v", x)

			//client := &Client{
			//	VMWareName: "Team 39 DB",
			//	VMWareId:   1313,
			//	MacAddress: "92:15:e7:7b:08:20",
			//	Team:       39,
			//	Group: Group{
			//		Name: "DB",
			//		Tags: []string{"db", "freebsd"},
			//	},
			//	CheckedIn: true,
			//}

			h, err := GetClientByMacAddress(x.MacAddress)
			if err != nil {
				c.String(500, "Unable to complete request: %+v", err)
			}

			c.JSON(200, h)

			return

			err = CommitClient(&h, s)
			if err != nil {
				c.String(500, "Unable to complete request: %+v", err)
			}
		})
	}

	return nil
}
