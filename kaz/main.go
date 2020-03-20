package kaz

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"log"
	"os"
	"time"
)

type Server struct {
	Port int
	Address string
	ReleaseMode string
	Engine *gin.Engine
	Logger *log.Logger
	Db *gorm.DB
}

const (
	Release string = gin.ReleaseMode
	Debug string = gin.DebugMode
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
	s.Engine = r

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
