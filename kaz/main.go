package kaz

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"gitlab.com/go-box/pongo2gin"
)

type Server struct {
	Port        int
	Address     string
	ReleaseMode string
	Engine      *gin.Engine
	Logger      *log.Logger
	Db          *gorm.DB
}

const (
	Release string = gin.ReleaseMode
	Debug   string = gin.DebugMode
)

func (s *Server) Run() {
	if s.Logger == nil {
		s.Logger = log.New(os.Stdout, "[KAZ] ", log.LstdFlags)
		s.Logger.Printf("LOGGER INITIALZIED")
	}

	l := s.Logger

	l.Printf("STARTUP TIME: %s", time.Now().String())
	gin.SetMode(s.ReleaseMode)

	r := gin.Default()
	r.HTMLRender = pongo2gin.Default()
	s.Engine = r

	r.Static("/static", "static")

	err := InitializeDatabase(s)
	if err != nil {
		l.Printf("RUNTIME ERROR: %+v", err)
	}

	s.Db.AutoMigrate(&Client{})

	err = routes(s)
	if err != nil {
		l.Printf("RUNTIME ERROR: %+v", err)
	}

	err = r.Run(fmt.Sprintf("%s:%d", s.Address, s.Port))
	if err != nil {
		l.Printf("RUNTIME ERROR: %+v", err)
	}
}
