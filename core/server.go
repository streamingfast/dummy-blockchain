package core

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/streamingfast/dummy-blockchain/types"
)

type Server struct {
	*gin.Engine

	store *Store
	addr  string
}

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func NewServer(store *Store, addr string) Server {
	server := Server{
		Engine: gin.Default(),
		store:  store,
		addr:   addr,
	}

	server.GET("/block", server.getBlock)
	server.GET("/blocks/:id", server.getBlock)

	return server
}

func (s *Server) Start() error {
	logrus.WithField("addr", s.addr).Info("starting server")
	err := s.Run(s.addr)
	if err != nil {
		logrus.WithError(err).Error("cant start server")
	}

	return err
}

func (s *Server) getBlock(c *gin.Context) {
	var (
		block *types.Block
		err   error
	)

	if id := c.Param("id"); len(id) > 0 {
		blockNum, err := strconv.Atoi(id)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}

		block, err = s.store.ReadBlock(uint64(blockNum))
	} else {
		block, err = s.store.CurrentBlock()
	}

	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	if block == nil {
		c.JSON(404, gin.H{"error": "block not found"})
		return
	}

	c.JSON(200, block)
}
